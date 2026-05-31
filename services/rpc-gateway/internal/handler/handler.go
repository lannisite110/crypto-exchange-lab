package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/crypto-exchange-lab/chainrpc"
	"github.com/crypto-exchange-lab/chainstore"
	apperrors "github.com/crypto-exchange-lab/go-common/errors"
	"github.com/crypto-exchange-lab/go-common/httputil"
)

// Handler serves multi-chain read APIs.
type Handler struct {
	Store *chainstore.Store
}

// New creates a handler.
func New(st *chainstore.Store) *Handler {
	return &Handler{Store: st}
}

// Register mounts routes on mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/chains", h.listChains)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/status", h.chainStatus)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/blocks", h.listBlocks)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/blocks/{number}", h.getBlock)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/blocks/{number}/transactions", h.blockTxs)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/transactions/{hash}", h.getTx)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/addresses/{addr}/transactions", h.addrTxs)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/addresses/{addr}/balance", h.addrBalance)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/events", h.listEvents)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/contracts", h.listContracts)
	mux.HandleFunc("GET /api/v1/chains/{chainId}/live/block/latest", h.liveLatestBlock)
}

func (h *Handler) listChains(w http.ResponseWriter, r *http.Request) {
	chains, err := h.Store.ListChains(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	httputil.OK(w, map[string]any{"chains": chains})
}

func (h *Handler) chainStatus(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	ch, err := h.Store.GetChain(r.Context(), chainID)
	if err != nil {
		fail(w, apperrors.New(apperrors.CodeNotFound, "chain not found"))
		return
	}
	sync, err := h.Store.GetSyncState(r.Context(), chainID)
	if err != nil {
		fail(w, err)
		return
	}
	head, err := h.liveHead(r.Context(), ch.RPCURL)
	lag := int64(0)
	if err == nil && head > uint64(sync.LastIndexedBlock) {
		lag = int64(head) - sync.LastIndexedBlock
	}
	httputil.OK(w, map[string]any{
		"chain":              ch,
		"last_indexed_block": sync.LastIndexedBlock,
		"live_head":          head,
		"lag_blocks":         lag,
		"updated_at":         sync.UpdatedAt,
	})
}

func (h *Handler) listBlocks(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	limit := queryInt(r, "limit", 20)
	blocks, err := h.Store.ListBlocks(r.Context(), chainID, limit)
	if err != nil {
		fail(w, err)
		return
	}
	httputil.OK(w, map[string]any{"chain_id": chainID, "blocks": blocks})
}

func (h *Handler) getBlock(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	num, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		fail(w, apperrors.New(apperrors.CodeInvalidArgument, "invalid block number"))
		return
	}
	blk, err := h.Store.GetBlock(r.Context(), chainID, num)
	if err != nil {
		fail(w, apperrors.New(apperrors.CodeNotFound, "block not found"))
		return
	}
	httputil.OK(w, blk)
}

func (h *Handler) blockTxs(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	num, err := strconv.ParseInt(r.PathValue("number"), 10, 64)
	if err != nil {
		fail(w, apperrors.New(apperrors.CodeInvalidArgument, "invalid block number"))
		return
	}
	txs, err := h.Store.ListBlockTransactions(r.Context(), chainID, num)
	if err != nil {
		fail(w, err)
		return
	}
	httputil.OK(w, map[string]any{"block_number": num, "transactions": txs})
}

func (h *Handler) getTx(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	hash := r.PathValue("hash")
	tx, err := h.Store.GetTransaction(r.Context(), chainID, hash)
	if err != nil {
		// fallback to live RPC
		ch, cerr := h.Store.GetChain(r.Context(), chainID)
		if cerr != nil {
			fail(w, apperrors.New(apperrors.CodeNotFound, "transaction not found"))
			return
		}
		live, lerr := h.fetchLiveTx(r.Context(), ch.RPCURL, hash)
		if lerr != nil {
			fail(w, apperrors.New(apperrors.CodeNotFound, "transaction not found"))
			return
		}
		httputil.OK(w, live)
		return
	}
	httputil.OK(w, tx)
}

func (h *Handler) addrTxs(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	addr := r.PathValue("addr")
	limit := queryInt(r, "limit", 20)
	txs, err := h.Store.ListTransactionsByAddress(r.Context(), chainID, addr, limit)
	if err != nil {
		fail(w, err)
		return
	}
	httputil.OK(w, map[string]any{"address": addr, "transactions": txs})
}

func (h *Handler) addrBalance(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	addr := r.PathValue("addr")
	ch, err := h.Store.GetChain(r.Context(), chainID)
	if err != nil {
		fail(w, apperrors.New(apperrors.CodeNotFound, "chain not found"))
		return
	}
	if !strings.HasPrefix(addr, "0x") {
		addr = "0x" + addr
	}
	bal, err := chainrpc.NewClient(ch.RPCURL).GetBalance(r.Context(), addr)
	if err != nil {
		fail(w, err)
		return
	}
	httputil.OK(w, map[string]any{
		"chain_id":   chainID,
		"address":    strings.ToLower(addr),
		"balance_wei": bal.String(),
	})
}

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	eventType := r.URL.Query().Get("type")
	limit := queryInt(r, "limit", 30)
	events, err := h.Store.ListEvents(r.Context(), chainID, eventType, limit)
	if err != nil {
		fail(w, err)
		return
	}
	httputil.OK(w, map[string]any{"chain_id": chainID, "events": events})
}

func (h *Handler) listContracts(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	contracts, err := h.Store.ListWatchedContracts(r.Context(), chainID)
	if err != nil {
		fail(w, err)
		return
	}
	httputil.OK(w, map[string]any{"chain_id": chainID, "contracts": contracts})
}

func (h *Handler) liveLatestBlock(w http.ResponseWriter, r *http.Request) {
	chainID := r.PathValue("chainId")
	ch, err := h.Store.GetChain(r.Context(), chainID)
	if err != nil {
		fail(w, apperrors.New(apperrors.CodeNotFound, "chain not found"))
		return
	}
	client := chainrpc.NewClient(ch.RPCURL)
	head, err := client.BlockNumber(r.Context())
	if err != nil {
		fail(w, err)
		return
	}
	blk, err := client.GetBlockByNumber(r.Context(), head, false)
	if err != nil {
		fail(w, err)
		return
	}
	ts, _ := chainrpc.BlockTimestampUnix(blk.Timestamp)
	httputil.OK(w, map[string]any{
		"number":     head,
		"hash":       blk.Hash,
		"parent_hash": blk.ParentHash,
		"timestamp":  ts,
		"tx_count":   len(blk.Transactions),
		"source":     "live_rpc",
	})
}

func (h *Handler) liveHead(ctx context.Context, rpcURL string) (uint64, error) {
	return chainrpc.NewClient(rpcURL).BlockNumber(ctx)
}

func (h *Handler) fetchLiveTx(ctx context.Context, rpcURL, hash string) (map[string]any, error) {
	if !strings.HasPrefix(hash, "0x") {
		hash = "0x" + hash
	}
	client := chainrpc.NewClient(rpcURL)
	tx, err := client.GetTransactionByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	bn, _ := chainrpc.BlockNumberHex(tx.BlockNumber)
	idx, _ := chainrpc.TxIndex(tx.TransactionIndex)
	out := map[string]any{
		"hash":         tx.Hash,
		"block_number": bn,
		"tx_index":     idx,
		"from_addr":    tx.From,
		"to_addr":      tx.To,
		"value_wei":    "0",
		"source":       "live_rpc",
	}
	if bi, err := chainrpc.HexToBigInt(tx.Value); err == nil {
		out["value_wei"] = bi.String()
	}
	rc, err := client.GetTransactionReceipt(ctx, hash)
	if err == nil {
		g, _ := chainrpc.BlockNumberHex(rc.GasUsed)
		st, _ := chainrpc.ReceiptStatus(rc.Status)
		out["gas_used"] = g
		out["status"] = st
	}
	return out, nil
}

func queryInt(r *http.Request, key string, def int) int {
	if v := r.URL.Query().Get(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func fail(w http.ResponseWriter, err error) {
	if ae, ok := err.(*apperrors.AppError); ok {
		httputil.Fail(w, ae)
		return
	}
	httputil.Fail(w, apperrors.New(apperrors.CodeInternal, err.Error()))
}
