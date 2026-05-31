package chainrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// Client calls Ethereum JSON-RPC over HTTP.
type Client struct {
	url    string
	http   *http.Client
	id     int
}

// NewClient creates an RPC client.
func NewClient(url string) *Client {
	return &Client{
		url: strings.TrimRight(url, "/"),
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (c *Client) call(ctx context.Context, method string, params []interface{}, out interface{}) error {
	c.id++
	body, err := json.Marshal(rpcRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      c.id,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var resp rpcResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("rpc %s: %s", method, resp.Error.Message)
	}
	if out == nil {
		return nil
	}
	if string(resp.Result) == "null" {
		return fmt.Errorf("rpc %s: null result", method)
	}
	return json.Unmarshal(resp.Result, out)
}

// BlockNumber returns the latest block number.
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
	var hexNum string
	if err := c.call(ctx, "eth_blockNumber", nil, &hexNum); err != nil {
		return 0, err
	}
	return hexToUint64(hexNum)
}

// GetBlockByNumber fetches a block; full=true includes transaction objects.
func (c *Client) GetBlockByNumber(ctx context.Context, num uint64, full bool) (*Block, error) {
	var blk Block
	param := fmt.Sprintf("0x%x", num)
	if err := c.call(ctx, "eth_getBlockByNumber", []interface{}{param, full}, &blk); err != nil {
		return nil, err
	}
	if blk.Hash == "" {
		return nil, fmt.Errorf("block %d not found", num)
	}
	return &blk, nil
}

// GetTransactionByHash loads a transaction.
func (c *Client) GetTransactionByHash(ctx context.Context, hash string) (*Transaction, error) {
	var tx Transaction
	if err := c.call(ctx, "eth_getTransactionByHash", []interface{}{hash}, &tx); err != nil {
		return nil, err
	}
	if tx.Hash == "" {
		return nil, fmt.Errorf("tx not found")
	}
	return &tx, nil
}

// GetTransactionReceipt loads a receipt.
func (c *Client) GetTransactionReceipt(ctx context.Context, hash string) (*Receipt, error) {
	var rc Receipt
	if err := c.call(ctx, "eth_getTransactionReceipt", []interface{}{hash}, &rc); err != nil {
		return nil, err
	}
	if rc.TransactionHash == "" {
		return nil, fmt.Errorf("receipt not found")
	}
	return &rc, nil
}

// GetLogs fetches logs for a filter.
func (c *Client) GetLogs(ctx context.Context, filter LogFilter) ([]Log, error) {
	var logs []Log
	if err := c.call(ctx, "eth_getLogs", []interface{}{filter}, &logs); err != nil {
		return nil, err
	}
	return logs, nil
}

// GetBalance returns account balance in wei.
func (c *Client) GetBalance(ctx context.Context, addr string) (*big.Int, error) {
	var hexBal string
	if err := c.call(ctx, "eth_getBalance", []interface{}{addr, "latest"}, &hexBal); err != nil {
		return nil, err
	}
	return hexToBigInt(hexBal)
}

// BlockNumberHex parses block.number field.
func BlockNumberHex(hexNum string) (uint64, error) {
	return hexToUint64(hexNum)
}

// BlockTimestampUnix parses block.timestamp.
func BlockTimestampUnix(hexTs string) (int64, error) {
	n, err := hexToUint64(hexTs)
	return int64(n), err
}

// TxIndex parses transaction index hex.
func TxIndex(hexIdx string) (int, error) {
	n, err := hexToUint64(hexIdx)
	return int(n), err
}

// LogIndexUint parses log index.
func LogIndexUint(hexIdx string) (int, error) {
	n, err := hexToUint64(hexIdx)
	return int(n), err
}

// ReceiptStatus parses receipt status (1 = success).
func ReceiptStatus(hexStatus string) (int, error) {
	n, err := hexToUint64(hexStatus)
	return int(n), err
}
