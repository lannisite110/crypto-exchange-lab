package chainrpc

import (
	"encoding/json"
	"math/big"
	"strings"
)

// Uniswap V2 Pair event topic0 hashes.
const (
	TopicSwap = "0xd78ad95fa46c994b655be0babfbacf937b37edd20e8186735c10e87b6dab95f0"
	TopicMint = "0x4c209b5fc8ad507833f3d9d4e9f268f54e4f284623fe92f254bd23f3106daf13"
	TopicBurn = "0xdccd412f0b943f337cfca98bacdbc111211dde3237e59ce95b53ad19113de21"
	TopicTransfer = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c94a06c563440acf7ed"
)

// ParsedEvent is a decoded log for storage/API.
type ParsedEvent struct {
	Type            string          `json:"type"`
	ContractAddress string          `json:"contract_address"`
	TxHash          string          `json:"tx_hash"`
	LogIndex        int             `json:"log_index"`
	BlockNumber     uint64          `json:"block_number"`
	Payload         json.RawMessage `json:"payload,omitempty"`
}

// ParseLog decodes known AMM / ERC20 logs.
func ParseLog(lg Log) (*ParsedEvent, error) {
	if len(lg.Topics) == 0 {
		return nil, nil
	}
	t0 := strings.ToLower(lg.Topics[0])
	blockNum, _ := hexToUint64(lg.BlockNumber)
	logIdx, _ := hexToUint64(lg.LogIndex)

	ev := &ParsedEvent{
		ContractAddress: strings.ToLower(lg.Address),
		TxHash:          lg.TxHash,
		LogIndex:        int(logIdx),
		BlockNumber:     blockNum,
	}

	switch t0 {
	case strings.ToLower(TopicSwap):
		ev.Type = "Swap"
		payload, err := decodeSwap(lg)
		if err != nil {
			return nil, err
		}
		ev.Payload = payload
	case strings.ToLower(TopicMint):
		ev.Type = "Mint"
		payload, err := decodeMintBurn(lg, "mint")
		if err != nil {
			return nil, err
		}
		ev.Payload = payload
	case strings.ToLower(TopicBurn):
		ev.Type = "Burn"
		payload, err := decodeMintBurn(lg, "burn")
		if err != nil {
			return nil, err
		}
		ev.Payload = payload
	case strings.ToLower(TopicTransfer):
		if len(lg.Topics) < 3 {
			return nil, nil
		}
		ev.Type = "Transfer"
		b, _ := json.Marshal(map[string]string{
			"from": "0x" + lg.Topics[1][26:],
			"to":   "0x" + lg.Topics[2][26:],
			"value": decodeUint256Data(lg.Data),
		})
		ev.Payload = b
	default:
		return nil, nil
	}
	return ev, nil
}

func decodeSwap(lg Log) (json.RawMessage, error) {
	data, err := decodeHexBytes(lg.Data)
	if err != nil {
		return nil, err
	}
	if len(data) < 128 {
		return nil, nil
	}
	out := map[string]string{
		"amount0_in":  wordToDecimal(data[0:32]),
		"amount1_in":  wordToDecimal(data[32:64]),
		"amount0_out": wordToDecimal(data[64:96]),
		"amount1_out": wordToDecimal(data[96:128]),
	}
	if len(lg.Topics) >= 2 {
		out["sender"] = "0x" + lg.Topics[1][26:]
	}
	if len(lg.Topics) >= 3 {
		out["to"] = "0x" + lg.Topics[2][26:]
	}
	b, err := json.Marshal(out)
	return b, err
}

func decodeMintBurn(lg Log, kind string) (json.RawMessage, error) {
	data, err := decodeHexBytes(lg.Data)
	if err != nil {
		return nil, err
	}
	out := map[string]string{"kind": kind}
	if len(data) >= 64 {
		out["amount0"] = wordToDecimal(data[0:32])
		out["amount1"] = wordToDecimal(data[32:64])
	}
	if len(lg.Topics) >= 2 {
		out["sender"] = "0x" + lg.Topics[1][26:]
	}
	b, err := json.Marshal(out)
	return b, err
}

func wordToDecimal(word []byte) string {
	bi := new(big.Int).SetBytes(word)
	return bi.String()
}

func decodeUint256Data(hexData string) string {
	b, err := decodeHexBytes(hexData)
	if err != nil || len(b) == 0 {
		return "0"
	}
	// pad to 32 bytes
	padded := make([]byte, 32)
	copy(padded[32-len(b):], b)
	return wordToDecimal(padded)
}
