package chainstore

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/crypto-exchange-lab/chainrpc"
)

// Syncer indexes blocks and AMM logs for one chain.
type Syncer struct {
	Store    *Store
	RPC      *chainrpc.Client
	ChainID  string
	Watch    []string
	BatchMax int
}

// RunOnce indexes up to BatchMax blocks from last cursor to chain head.
func (sy *Syncer) RunOnce(ctx context.Context) (int, error) {
	st, err := sy.Store.GetSyncState(ctx, sy.ChainID)
	if err != nil {
		return 0, err
	}
	head, err := sy.RPC.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	start := uint64(st.LastIndexedBlock) + 1
	if start == 0 {
		// first run: stay a few blocks behind head for stability
		if head > 3 {
			start = head - 3
		}
	}
	if start > head {
		return 0, nil
	}
	end := start + uint64(sy.BatchMax) - 1
	if end > head {
		end = head
	}
	indexed := 0
	for n := start; n <= end; n++ {
		if err := sy.indexBlock(ctx, n); err != nil {
			return indexed, fmt.Errorf("block %d: %w", n, err)
		}
		if err := sy.Store.SetSyncState(ctx, sy.ChainID, int64(n)); err != nil {
			return indexed, err
		}
		indexed++
	}
	return indexed, nil
}

func (sy *Syncer) indexBlock(ctx context.Context, num uint64) error {
	blk, err := sy.RPC.GetBlockByNumber(ctx, num, true)
	if err != nil {
		return err
	}
	bn, _ := chainrpc.BlockNumberHex(blk.Number)
	ts, _ := chainrpc.BlockTimestampUnix(blk.Timestamp)

	if err := sy.Store.InsertBlock(ctx, Block{
		ChainID:    sy.ChainID,
		Number:     int64(bn),
		Hash:       blk.Hash,
		ParentHash: blk.ParentHash,
		Timestamp:  time.Unix(ts, 0).UTC(),
		TxCount:    len(blk.Transactions),
	}); err != nil {
		return err
	}

	for i, raw := range blk.Transactions {
		txMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		hash, _ := txMap["hash"].(string)
		from, _ := txMap["from"].(string)
		to, _ := txMap["to"].(string)
		val, _ := txMap["value"].(string)
		valWei := "0"
		if bi, err := chainrpcHexBig(val); err == nil {
			valWei = bi.String()
		}
		var gasUsed *int64
		var status *int
		if hash != "" {
			rc, err := sy.RPC.GetTransactionReceipt(ctx, hash)
			if err == nil {
				g, _ := chainrpc.BlockNumberHex(rc.GasUsed)
				gu := int64(g)
				gasUsed = &gu
				st, _ := chainrpc.ReceiptStatus(rc.Status)
				status = &st
				for _, lg := range rc.Logs {
					if err := sy.indexLog(ctx, lg, int64(bn)); err != nil {
						return err
					}
				}
			}
		}
		if err := sy.Store.InsertTransaction(ctx, Transaction{
			ChainID:     sy.ChainID,
			Hash:        hash,
			BlockNumber: int64(bn),
			TxIndex:     i,
			FromAddr:    from,
			ToAddr:      to,
			ValueWei:    valWei,
			GasUsed:     gasUsed,
			Status:      status,
		}); err != nil {
			return err
		}
	}

	if len(sy.Watch) > 0 {
		filter := chainrpc.LogFilter{
			FromBlock: fmt.Sprintf("0x%x", num),
			ToBlock:   fmt.Sprintf("0x%x", num),
			Address:   sy.Watch,
		}
		logs, err := sy.RPC.GetLogs(ctx, filter)
		if err == nil {
			for _, lg := range logs {
				if err := sy.indexLog(ctx, lg, int64(bn)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (sy *Syncer) indexLog(ctx context.Context, lg chainrpc.Log, blockNum int64) error {
	parsed, err := chainrpc.ParseLog(lg)
	if err != nil || parsed == nil {
		return err
	}
	topic0 := ""
	if len(lg.Topics) > 0 {
		topic0 = lg.Topics[0]
	}
	return sy.Store.InsertEvent(ctx, Event{
		ChainID:         sy.ChainID,
		BlockNumber:     blockNum,
		TxHash:          lg.TxHash,
		LogIndex:        parsed.LogIndex,
		ContractAddress: parsed.ContractAddress,
		EventType:       parsed.Type,
		Payload:         parsed.Payload,
	}, topic0, lg.Data)
}

func chainrpcHexBig(h string) (*big.Int, error) {
	h = strings.TrimPrefix(strings.TrimPrefix(h, "0x"), "0X")
	if h == "" {
		return big.NewInt(0), nil
	}
	bi := new(big.Int)
	if _, ok := bi.SetString(h, 16); !ok {
		return nil, fmt.Errorf("invalid hex")
	}
	return bi, nil
}

// DecodePayload helper for API layers.
func DecodePayload(raw json.RawMessage, out interface{}) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, out)
}
