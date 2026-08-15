package ledger

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"time"
)

type Block struct {
	Index      uint64
	Timestamp  time.Time
	MerkleRoot []byte
	PrevHash   []byte
	Hash       []byte
}

func calculateHash(index uint64, prevHash, merkleRoot []byte, ts time.Time) []byte {
	h := sha256.New()
	buf := make([]byte, 8)

	binary.BigEndian.PutUint64(buf, index)
	h.Write(buf)

	h.Write(prevHash)
	h.Write(merkleRoot)

	binary.BigEndian.PutUint64(buf, uint64(ts.UnixNano()))
	h.Write(buf)

	return h.Sum(nil)
}

type Chain struct {
	mu     sync.RWMutex
	blocks []Block
}

func NewChain() *Chain {
	return &Chain{
		blocks: make([]Block, 0),
	}
}

func (c *Chain) Append(root []byte, ts time.Time) Block {
	c.mu.Lock()
	defer c.mu.Unlock()

	var prevHash []byte
	var index uint64

	if len(c.blocks) == 0 {
		prevHash = make([]byte, 32)
		index = 0
	} else {
		prevBlock := c.blocks[len(c.blocks)-1]
		prevHash = prevBlock.Hash
		index = prevBlock.Index + 1
	}

	hash := calculateHash(index, prevHash, root, ts)

	block := Block{
		Index:      index,
		Timestamp:  ts,
		MerkleRoot: root,
		PrevHash:   prevHash,
		Hash:       hash,
	}

	c.blocks = append(c.blocks, block)
	return block
}

func (c *Chain) Validate() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.blocks) == 0 {
		return true
	}

	zeroHash := make([]byte, 32)

	for i, block := range c.blocks {

		expectedHash := calculateHash(block.Index, block.PrevHash, block.MerkleRoot, block.Timestamp)
		if !bytes.Equal(block.Hash, expectedHash) {
			return false
		}

		if i == 0 {
			if !bytes.Equal(block.PrevHash, zeroHash) {
				return false
			}
		} else {
			if !bytes.Equal(block.PrevHash, c.blocks[i-1].Hash) {
				return false
			}
		}

	}

	return true
}

func (c *Chain) TipHash() ([]byte, int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if len(c.blocks) == 0 {
		return nil, 0
	}
	return c.blocks[len(c.blocks)-1].Hash, len(c.blocks)
}
