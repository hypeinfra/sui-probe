package sui

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const OfficialDevNode = "fullnode.devnet.sui.io"

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
			"User-Agent":   []string{"SUI-HealthCheck/0.0.0"},
		},
	}
}

type GetTotalTransactionNumberStruct struct {
	JSONRPC string `json:"jsonrpc"`
	Result  int    `json:"result"`
}

func (n *NodeClient) GetTotalTransactionNumber() (int, error) {
	message := bytes.NewBufferString(`{"jsonrpc": "2.0","method": "sui_getTotalTransactionNumber","id": 1}`)
	values := &GetTotalTransactionNumberStruct{}
	RPCError := &JSONRPCError{}
	parsedURL, err := url.Parse(n.Address)
	if err != nil {
		return 0, err
	}
	do, err := n.Client.Do(&http.Request{
		Method: "POST",
		URL:    parsedURL,
		Header: *n.Header,
		Body:   io.NopCloser(message),
	})
	if err != nil {
		return 0, err
	}
	if do.StatusCode != http.StatusOK {
		err = json.NewDecoder(do.Body).Decode(&RPCError)
		if err != nil {
			return 0, err
		}
		return 0, RPCError
	}
	err = json.NewDecoder(do.Body).Decode(values)
	if err != nil {
		return 0, err
	}
	return values.Result, nil
}
