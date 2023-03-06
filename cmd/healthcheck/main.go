package main

import (
	"fmt"
	"github.com/Jeffail/gabs/v2"
	"github.com/hypeinfra/sui-probe/static"
	"github.com/hypeinfra/sui-probe/sui"
	"github.com/hypeinfra/sui-probe/templates"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/errgroup"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/netip"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

const HTMLNodeLoadHead = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="X-UA-Compatible" content="IE=edge">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="stylesheet" href="/static/main.css">
  <title>Project head</title>
</head>
<body>
<div id="progress">Loading node info`

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data interface{}, _ echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

type NodeInfo struct {
	Transactions  uint
	Version       string
	SchemasAmount uint
	MethodsAmount uint
	RawContainer  *gabs.Container
}

func GatherNodeInfo(node *sui.NodeClient, result *NodeInfo) func() error {
	return func() error {
		transactions, err := node.GetTotalTransactionNumber()
		if err != nil {
			return err
		}
		result.Transactions = uint(transactions)

		nodeInfo, err := node.Discover()
		if err != nil {
			return err
		}

		result.RawContainer, err = gabs.ParseJSON(nodeInfo)
		if err != nil {
			return err
		}

		result.Version = result.RawContainer.Path("result.info.version").Data().(string)
		result.SchemasAmount = uint(len(result.RawContainer.Path("result.components.schemas").Children()))
		result.MethodsAmount = uint(len(result.RawContainer.Path("result.methods").Children()))

		return nil
	}
}

func main() {
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.LoggerWithConfig(
		middleware.LoggerConfig{
			Format:           `${time_custom} [Echo] ${latency_human} ${method} "${uri}" from ${remote_ip} "${user_agent}" | Error="${error}" ` + "\n",
			CustomTimeFormat: "2006/01/02 15:04:05",
		}))
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())

	t := template.Must(template.ParseFS(templates.Templates, "*.gohtml", "*/*.gohtml"))
	e.Renderer = &Template{
		t,
	}

	e.GET("/troubleshooting", func(c echo.Context) error {
		return c.Render(http.StatusOK, "troubleshooting.gohtml", map[string]any{"Title": "Troubleshooting Sui Node"})
	})

	e.GET("/", func(c echo.Context) error {
		nodeIP := c.QueryParam("sui-node-address")
		if nodeIP != "" {
			ipaddr, err := netip.ParseAddrPort(nodeIP)
			if err != nil {
				if err.Error() == "not an ip:port" {
					return c.Render(http.StatusOK, "index.gohtml", map[string]any{"error": "Did you forgot to specify the port? Check the format is correct. For example: 127.0.0.1:9000", "ip": nodeIP})
				}
				return c.Render(http.StatusOK, "index.gohtml", map[string]any{"error": err.Error(), "ip": nodeIP})
			}
			// TODO: add flag for allowing private address space
			if ipaddr.Addr().IsPrivate() {
				return c.Render(http.StatusOK, "index.gohtml", map[string]any{"error": "private address space is disabled on this instance", "ip": nodeIP})
			}

			userNode := sui.NewNode("http://" + ipaddr.String())
			officialNode := sui.NewNode("https://" + sui.OfficialDevNode)
			userNodeMetrics := sui.NewMetricsClient("http://" + ipaddr.Addr().String() + ":9184")

			g := new(errgroup.Group)

			var (
				officialNodeInfo, providedNodeInfo, providedNodeInfoWithSleep, officialNodeInfoWithSleep NodeInfo
				uptimeDuration, totalEpochDuration                                                       time.Duration
				providedNodePeers, currentEpoch, currentVotingRight                                      uint64
				metricsNotAvailable                                                                      bool
			)

			g.Go(func() error {
				_ = userNodeMetrics.GetMetrics()
				uptime, _ := userNodeMetrics.GetUptime()
				// if there is no uptime, then we can't get any metrics
				if uptime == "" {
					metricsNotAvailable = true
					return nil
				}
				uptimeDuration, _ = time.ParseDuration(uptime + "s")
				providedNodePeers, _ = userNodeMetrics.GetPeers()
				currentEpoch, _ = userNodeMetrics.GetCurrentEpoch()
				totalEpochDuration, _ = userNodeMetrics.GetTotalEpochDuration()
				currentVotingRight, _ = userNodeMetrics.GetCurrentVotingRight()
				return nil
			})

			g.Go(GatherNodeInfo(officialNode, &officialNodeInfo))
			g.Go(GatherNodeInfo(userNode, &providedNodeInfo))

			err = g.Wait()
			if err != nil && strings.Contains(err.Error(), "context deadline exceeded") {
				return c.Render(http.StatusOK, "index.gohtml", map[string]any{"error": "Node timeout. Check if your node is running, your firewall rules and node's logs.", "ip": nodeIP})
			} else if err != nil && strings.Contains(err.Error(), "machine actively refused it.") {
				return c.Render(http.StatusOK, "index.gohtml", map[string]any{"error": "Your node is actively rejecting the connection. Perhaps you are using a different port or have forgotten to add rules to your firewall?", "ip": nodeIP})
			} else if err != nil {
				return c.Render(http.StatusOK, "index.gohtml", map[string]any{"error": err.Error(), "ip": nodeIP})
			}

			// Loading message
			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
			writeLoadingMessageOnce := sync.Once{}
			for i := 0; i < 3; i++ {
				// Write to stream that we are loading info
				writeLoadingMessageOnce.Do(func() {
					// Start of HTML document
					_, _ = c.Response().Write([]byte(HTMLNodeLoadHead))
				})
				_, _ = c.Response().Write([]byte("."))
				c.Response().Flush()
				time.Sleep(1 * time.Second)
			}

			// Gather info after 3 seconds
			g.Go(GatherNodeInfo(userNode, &providedNodeInfoWithSleep))
			g.Go(GatherNodeInfo(officialNode, &officialNodeInfoWithSleep))

			// Write to stream that we are loading info
			g.Go(func() error {
				_, err = c.Response().Write([]byte("."))
				c.Response().Flush()
				time.Sleep(1 * time.Second)
				return err
			})
			err = g.Wait()

			// End of our HTML, so we can hide those dots
			_, _ = c.Response().Write([]byte("</div>"))
			_, _ = c.Response().Write([]byte("<style>#progress { display: none; }</style>"))
			c.Response().Flush()

			if err != nil {
				return c.Render(http.StatusOK, "index.gohtml", map[string]any{"error": err.Error(), "ip": nodeIP})
			}

			ProvidedNodeTPS := providedNodeInfoWithSleep.Transactions - providedNodeInfo.Transactions
			OfficialNodeTPS := officialNodeInfoWithSleep.Transactions - officialNodeInfo.Transactions
			syncStatusInPercents := float64(providedNodeInfo.Transactions) / float64(officialNodeInfo.Transactions) * 100
			CanProvidedNodeCatchUp := (ProvidedNodeTPS >= OfficialNodeTPS || syncStatusInPercents > 95) && (ProvidedNodeTPS != 0 || OfficialNodeTPS != 0)
			// If transactions amount is more than on official node, then something is wrong
			syncTransactionsInvalid := providedNodeInfo.Transactions > officialNodeInfo.Transactions
			syncPredictedTimeWait := time.Duration(float64(officialNodeInfo.Transactions-providedNodeInfo.Transactions)/float64(ProvidedNodeTPS)) * time.Second
			isProvidedNodeOutdated := officialNodeInfo.Version != providedNodeInfo.Version
			syncZeroSpeedCheck := ProvidedNodeTPS == 0 && providedNodeInfo.Transactions != officialNodeInfo.Transactions

			if metricsNotAvailable {
				return c.Render(http.StatusOK, "node.gohtml", map[string]any{
					"Title":                       "Node metrics",
					"ip":                          nodeIP,
					"transactions":                providedNodeInfo.Transactions,
					"transactionsOfficial":        officialNodeInfo.Transactions,
					"version":                     providedNodeInfo.Version,
					"versionOfficial":             officialNodeInfo.Version,
					"schemasAmount":               providedNodeInfo.SchemasAmount,
					"schemasAmountOfficial":       officialNodeInfo.SchemasAmount,
					"methodsAmount":               providedNodeInfo.MethodsAmount,
					"methodsAmountOfficial":       officialNodeInfo.MethodsAmount,
					"NodeTPS":                     ProvidedNodeTPS,
					"OfficialNodeTPS":             OfficialNodeTPS,
					"CanProvidedNodeCatchUp":      CanProvidedNodeCatchUp,
					"NodeOutdated":                isProvidedNodeOutdated,
					"NodeSyncStatus":              fmt.Sprintf("%.2f", syncStatusInPercents) + "%",
					"NodeSyncTimeWait":            syncPredictedTimeWait,
					"NodeSyncTransactionsInvalid": syncTransactionsInvalid,
					"NoStats":                     metricsNotAvailable,
				})
			}

			return c.Render(http.StatusOK, "node.gohtml", map[string]any{
				"Title":                       "Node metrics",
				"ip":                          nodeIP,
				"transactions":                providedNodeInfo.Transactions,
				"transactionsOfficial":        officialNodeInfo.Transactions,
				"version":                     providedNodeInfo.Version,
				"versionOfficial":             officialNodeInfo.Version,
				"schemasAmount":               providedNodeInfo.SchemasAmount,
				"schemasAmountOfficial":       officialNodeInfo.SchemasAmount,
				"methodsAmount":               providedNodeInfo.MethodsAmount,
				"methodsAmountOfficial":       officialNodeInfo.MethodsAmount,
				"NodeTPS":                     ProvidedNodeTPS,
				"OfficialNodeTPS":             OfficialNodeTPS,
				"CanProvidedNodeCatchUp":      CanProvidedNodeCatchUp,
				"NodeOutdated":                isProvidedNodeOutdated,
				"NodeSyncStatus":              fmt.Sprintf("%.2f", syncStatusInPercents) + "%",
				"NodeSyncTimeWait":            syncPredictedTimeWait,
				"NodeSyncTransactionsInvalid": syncTransactionsInvalid,
				"NoStats":                     metricsNotAvailable,
				"NodeUptime":                  uptimeDuration,
				"NodePeers":                   providedNodePeers,
				"NodeSyncZeroSpeedCheck":      syncZeroSpeedCheck,
				"NodeCurrentEpoch":            currentEpoch,
				"NodeTotalEpochDuration":      totalEpochDuration,
				"NodeCurrentVotingRight":      currentVotingRight,
			})
		}
		return c.Render(http.StatusOK, "index.gohtml", nil)
	})
	e.StaticFS("/static", static.FS)

	go func() {
		sig := <-signalChannel
		switch sig {
		case os.Interrupt, syscall.SIGTERM:
			log.Println("[Exit signal] Shutting down HTTP server")
			signal.Stop(signalChannel)
			err := e.Close()
			if err != nil {
				log.Fatalln("An error occurred while trying to shutdown the server, fatal:", err)
			}
		}
	}()

	err := e.Start(":1323")
	if err != nil && err != http.ErrServerClosed {
		log.Fatalln("A server encountered an error, fatal:", err)
	}
}
