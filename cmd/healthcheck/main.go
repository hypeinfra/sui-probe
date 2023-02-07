package main

import (
	"github.com/Jeffail/gabs/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/sync/errgroup"
	"html/template"
	"io"
	"net/http"
	"net/netip"
	"sui/static"
	"sui/sui"
	"sui/templates"
	"sync"
	"time"
)

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

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Gzip())
	t := template.Must(template.ParseFS(templates.Templates, "*.gohtml", "*/*.gohtml"))
	e.Renderer = &Template{
		t,
	}
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

			g := new(errgroup.Group)

			var (
				officialNodeInfo, providedNodeInfo, providedNodeInfoWithSleep NodeInfo
			)

			g.Go(GatherNodeInfo(officialNode, &officialNodeInfo))
			g.Go(GatherNodeInfo(userNode, &providedNodeInfo))

			err = g.Wait()
			if err != nil {
				return c.Render(http.StatusOK, "index.gohtml", map[string]any{"error": err.Error(), "ip": nodeIP})
			}

			c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
			writeLoadingMessageOnce := sync.Once{}
			for i := 0; i < 4; i++ {
				// Write to stream that we are loading info
				writeLoadingMessageOnce.Do(func() {
					// <!DOCTYPE html>
					// <html lang="en">
					// <head>
					//    <meta charset="UTF-8">
					//    <meta http-equiv="X-UA-Compatible" content="IE=edge">
					//    <meta name="viewport" content="width=device-width, initial-scale=1.0">
					//    <link rel="stylesheet" href="/static/main.css">
					//    <title>Project head</title>
					// </head>
					_, _ = c.Response().Write([]byte(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta http-equiv="X-UA-Compatible" content="IE=edge">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <link rel="stylesheet" href="/static/main.css">
  <title>Project head</title>
</head>
<body>
<div id="progress">Loading node info`))
				})
				_, _ = c.Response().Write([]byte("."))
				c.Response().Flush()
				time.Sleep(1 * time.Second)
			}
			_, _ = c.Response().Write([]byte("</div>"))
			_, _ = c.Response().Write([]byte("<style>#progress { display: none; }</style>"))
			c.Response().Flush()
			g.Go(GatherNodeInfo(userNode, &providedNodeInfoWithSleep))

			return c.Render(http.StatusOK, "node.gohtml", map[string]any{
				"ip":                    nodeIP,
				"transactions":          providedNodeInfo.Transactions,
				"transactionsOfficial":  officialNodeInfo.Transactions,
				"version":               providedNodeInfo.Version,
				"versionOfficial":       officialNodeInfo.Version,
				"schemasAmount":         providedNodeInfo.SchemasAmount,
				"schemasAmountOfficial": officialNodeInfo.SchemasAmount,
				"methodsAmount":         providedNodeInfo.MethodsAmount,
				"methodsAmountOfficial": officialNodeInfo.MethodsAmount,
			})
		}
		return c.Render(http.StatusOK, "index.gohtml", nil)
	})
	e.StaticFS("/static", static.FS)
	e.Logger.Fatal(e.Start(":1323"))
}
