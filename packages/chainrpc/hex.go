package chainrpc

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

func hexToUint64(h string) (uint64, error) {
	h = strings.TrimPrefix(strings.TrimPrefix(h, "0x"), "0X")
	if h == "" {
		return 0, nil
	}
	var n uint64
	_, err := fmt.Sscanf(h, "%x", &n)
	if err != nil {
		// large hex
		bi := new(big.Int)
		if _, ok := bi.SetString(h, 16); !ok {
			return 0, fmt.Errorf("invalid hex: %s", h)
		}
		if !bi.IsUint64() {
			return 0, fmt.Errorf("overflow: %s", h)
		}
		return bi.Uint64(), nil
	}
	return n, nil
}

// HexToBigInt parses a 0x-prefixed hex quantity.
func HexToBigInt(h string) (*big.Int, error) {
	return hexToBigInt(h)
}

func hexToBigInt(h string) (*big.Int, error) {
	h = strings.TrimPrefix(strings.TrimPrefix(h, "0x"), "0X")
	if h == "" {
		return big.NewInt(0), nil
	}
	bi := new(big.Int)
	if _, ok := bi.SetString(h, 16); !ok {
		return nil, fmt.Errorf("invalid hex: %s", h)
	}
	return bi, nil
}

func decodeHexBytes(h string) ([]byte, error) {
	h = strings.TrimPrefix(strings.TrimPrefix(h, "0x"), "0X")
	if len(h)%2 == 1 {
		h = "0" + h
	}
	return hex.DecodeString(h)
}
