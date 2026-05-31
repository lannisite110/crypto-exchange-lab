package chainrpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBlockNumber(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"result":  "0x2a",
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	n, err := c.BlockNumber(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if n != 42 {
		t.Fatalf("got %d", n)
	}
}

func TestParseSwapLog(t *testing.T) {
	lg := Log{
		Address:     "0xabc",
		Topics:      []string{TopicSwap, "0x0000000000000000000000001111111111111111111111111111111111111111"},
		Data:        "0x" + repeatHex(64) + repeatHex(64) + repeatHex(64) + repeatHex(64),
		BlockNumber: "0x10",
		TxHash:      "0xdef",
		LogIndex:    "0x0",
	}
	ev, err := ParseLog(lg)
	if err != nil {
		t.Fatal(err)
	}
	if ev == nil || ev.Type != "Swap" {
		t.Fatalf("unexpected %v", ev)
	}
}

func repeatHex(n int) string {
	out := make([]byte, n)
	for i := range out {
		out[i] = '1'
	}
	return string(out)
}
