package server

import (
	"encoding/pem"
	"fmt"
)

// pemDecode wraps pem.Decode for use in handlers.
func pemDecode(data string) (*pem.Block, error) {
	block, _ := pem.Decode([]byte(data))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM data")
	}
	return block, nil
}

// pemDecodeSimple returns just the block without error (for compatibility).
func pemDecodeSimple(data string) *pem.Block {
	block, _ := pem.Decode([]byte(data))
	return block
}

// errToString converts an error to string, returning "" for nil.
func errToString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
