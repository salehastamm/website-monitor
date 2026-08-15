package ledger

import (
	"bytes"
	"testing"
	"time"
)

func TestGenesisBlock(t *testing.T) {
	chain := NewChain()
	ts := time.Now()
	root := []byte("root1")

	block := chain.Append(root, ts)
	expectedZero := make([]byte, 32)
	if !bytes.Equal(block.PrevHash, expectedZero) {
		t.Errorf("Genesis block PrevHash should be zeros, got %x", block.PrevHash)
	}

	if block.Index != 0 {
		t.Errorf("Genesis block Index should be 0, got %d", block.Index)
	}

	expectedHash := calculateHash(0, expectedZero, root, ts)
	if !bytes.Equal(block.Hash, expectedHash) {
		t.Errorf("Genesis block Hash was not calculated correctly")
	}
}

func TestValidateValidChain(t *testing.T) {
	chain := NewChain()
	ts := time.Now()
	chain.Append([]byte("root1"), ts)
	chain.Append([]byte("root2"), ts.Add(time.Second))
	chain.Append([]byte("root1"), ts.Add(2*time.Second))

	if !chain.Validate() {
		t.Errorf("A freshly built chain should be valid")
	}
}

func TestTamperedMerkleRoot(t *testing.T) {
	chain := NewChain()
	ts := time.Now()
	chain.Append([]byte("root1"), ts)
	chain.Append([]byte("root2"), ts.Add(time.Second))

	chain.blocks[0].MerkleRoot = []byte("tampered")

	if chain.Validate() {
		t.Errorf("Validation should fail if a MerkleRoot was mutated")
	}
}

func TestTamperedPrevHash(t *testing.T) {
	chain := NewChain()
	ts := time.Now()
	chain.Append([]byte("root1"), ts)
	chain.Append([]byte("root2"), ts.Add(time.Second))

	chain.blocks[1].PrevHash = make([]byte, 32)

	if chain.Validate() {
		t.Errorf("Validation should fail if PrevHash was mutated")
	}
}

func TestRecomputeDeterminism(t *testing.T) {
	// using a constant time for estimating determinism
	ts := time.Unix(1672531200, 0)
	root := []byte("fixed_root_data")
	prevHash := make([]byte, 32)
	index := uint64(0)

	expectedHash := calculateHash(index, prevHash, root, ts)

	chain := NewChain()
	block := chain.Append(root, ts)

	if !bytes.Equal(block.Hash, expectedHash) {
		t.Errorf("Calculated hash does not match independent recomputation")
	}
}

func TestSameInputsProduceSameTip(t *testing.T) {
	ts := time.Unix(1672531200, 0)
	root1 := []byte("rootA")
	root2 := []byte("rootB")

	chain1 := NewChain()
	chain1.Append(root1, ts)
	chain1.Append(root2, ts)

	chain2 := NewChain()
	chain2.Append(root1, ts)
	chain2.Append(root2, ts)

	tip1, _ := chain1.TipHash()
	tip2, _ := chain2.TipHash()

	if !bytes.Equal(tip1, tip2) {
		t.Errorf("Identical chains should produce identical tip hashes")
	}
}
