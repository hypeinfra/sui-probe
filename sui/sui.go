package sui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const OfficialDevNode = "fullnode.devnet.sui.io"
const OfficialTestNode = "fullnode.testnet.sui.io"

type NodeClient struct {
	Address string
	Client  *http.Client
	Header  *http.Header
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (j *JSONRPCError) Error() string {
	return "JSON RPC Error: " + j.Message + " (Code: " + strconv.Itoa(j.Code) + ")"
}

func NewNode(address string) *NodeClient {
	return &NodeClient{
		Address: address,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		Header: &http.Header{
			"Content-Type": []string{"application/json"},
			"User-Agent":   []string{"SUI-HealthCheck/0.0.4 (+https://github.com/hypeinfra/sui-probe)"},
		},
	}
}

type GetTotalTransactionNumberStruct struct {
	JSONRPC string `json:"jsonrpc"`
	Result  int    `json:"result"`
}

func (n *NodeClient) GetTotalTransactionNumber() (int, error) {
	message := bytes.NewBufferString(`{"jsonrpc": "2.0","method": "sui_getTotalTransactionNumber","id": 69}`)
	values := &GetTotalTransactionNumberStruct{}
	rpcError := &JSONRPCError{}
	parsedURL, err := url.Parse(n.Address)
	if err != nil {
		return 0, fmt.Errorf("could not parse node address: %w", err)
	}
	do, err := n.Client.Do(&http.Request{
		Method: "POST",
		URL:    parsedURL,
		Header: *n.Header,
		Body:   io.NopCloser(message),
	})
	if err != nil {
		return 0, fmt.Errorf("http request to the node failed: %w", err)
	}

	data, err := io.ReadAll(do.Body)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(data)

	if do.StatusCode != http.StatusOK {
		err = json.NewDecoder(buf).Decode(&rpcError)
		if err != nil {
			return 0, fmt.Errorf("error parsing failed, no RPC error was provided: %w. Response from the node: %q", err, buf)
		}
		return 0, rpcError
	}
	err = json.NewDecoder(buf).Decode(values)
	if err != nil {
		err = json.NewDecoder(buf).Decode(&rpcError)
		if err != nil {
			return 0, fmt.Errorf("node responded with OK HTTP status, but program failed to decode the response, no RPC error was provided: %w. Response from the node: %q", err, buf)
		}
		return 0, fmt.Errorf("node responded with OK HTTP status, but program failed to decode RPC response: %w. Response from the node: %q", err, buf)
	}
	return values.Result, nil
}

func (n *NodeClient) Discover() (json.RawMessage, error) {
	message := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"rpc.discover","id":69}`)
	rpcError := &JSONRPCError{}
	parsedURL, err := url.Parse(n.Address)
	if err != nil {
		return nil, err
	}
	do, err := n.Client.Do(&http.Request{
		Method: "POST",
		URL:    parsedURL,
		Header: *n.Header,
		Body:   io.NopCloser(message),
	})
	if err != nil {
		return nil, err
	}
	if do.StatusCode != http.StatusOK {
		err = json.NewDecoder(do.Body).Decode(&rpcError)
		if err != nil {
			return nil, err
		}
		return nil, rpcError
	}
	var values json.RawMessage
	err = json.NewDecoder(do.Body).Decode(&values)
	if err != nil {
		return nil, err
	}
	return values, nil
}
