package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type HashBlock struct {
	Index     int       `json:"index"`
	PrevHash  string    `json:"prev_hash"`
	DataHash  string    `json:"data_hash"`
	Timestamp time.Time `json:"timestamp"`
}

type HashChain struct {
	mu     sync.RWMutex
	blocks []HashBlock
}

func NewHashChain() *HashChain {
	return &HashChain{
		blocks: []HashBlock{
			{Index: 0, PrevHash: "0000000000000000000000000000000000000000000000000000000000000000", DataHash: "genesis", Timestamp: time.Now()},
		},
	}
}

func (hc *HashChain) AppendBlock(event any) *HashBlock {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	last := hc.blocks[len(hc.blocks)-1]
	data, _ := json.Marshal(event)
	dataHash := hashData(data)
	block := HashBlock{
		Index:     last.Index + 1,
		PrevHash:  computeBlockHash(last),
		DataHash:  dataHash,
		Timestamp: time.Now(),
	}
	hc.blocks = append(hc.blocks, block)
	return &block
}

func (hc *HashChain) VerifyChain() bool {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	for i := 1; i < len(hc.blocks); i++ {
		block := hc.blocks[i]
		prevBlock := hc.blocks[i-1]
		expectedPrevHash := computeBlockHash(prevBlock)
		if block.PrevHash != expectedPrevHash {
			return false
		}
	}
	return true
}

func (hc *HashChain) GetChainProof(fromIndex, toIndex int) []HashBlock {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	if fromIndex < 0 || toIndex >= len(hc.blocks) || fromIndex > toIndex {
		return nil
	}
	proof := make([]HashBlock, toIndex-fromIndex+1)
	copy(proof, hc.blocks[fromIndex:toIndex+1])
	return proof
}

func (hc *HashChain) DetectTamper() (int, bool) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	for i := 1; i < len(hc.blocks); i++ {
		block := hc.blocks[i]
		prevBlock := hc.blocks[i-1]
		expectedPrevHash := computeBlockHash(prevBlock)
		if block.PrevHash != expectedPrevHash {
			return block.Index, true
		}
	}
	return -1, false
}

func (hc *HashChain) Length() int {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return len(hc.blocks)
}

func computeBlockHash(b HashBlock) string {
	data := fmt.Sprintf("%d:%s:%s:%d", b.Index, b.PrevHash, b.DataHash, b.Timestamp.UnixNano())
	return hashData([]byte(data))
}

func hashData(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}