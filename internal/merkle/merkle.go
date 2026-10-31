package merkle

import (
	"bytes"
	"crypto/sha256"
	"sort"
)

func hashLeaf(data []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x00})
	h.Write(data)
	result := h.Sum(nil)
	return result
}

func hashNode(left, right []byte) []byte {
	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(left)
	h.Write(right)
	result := h.Sum(nil)
	return result
}

func Root(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		return nil
	}
	// first layer : Hash the leaves
	var level [][]byte // a list of HASHs
	for _, leaf := range leaves {
		level = append(level, hashLeaf(leaf))
	}
	// go up layer by layer so one element remains
	for len(level) > 1 {
		var next [][]byte
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				next = append(next, hashNode(level[i], level[i+1]))
			} else {
				next = append(next, level[i])
			}
		}
		level = next
	}

	return level[0]
}

type ProofStep struct {
	Hash   []byte
	IsLeft bool
}

func Verify(leaf []byte, proof []ProofStep, root []byte) bool {
	current := hashLeaf(leaf)

	for _, step := range proof {
		if step.IsLeft {
			current = hashNode(step.Hash, current)
		} else {
			current = hashNode(current, step.Hash)
		}
	}

	return bytes.Equal(current, root)
}

func Proof(leaves [][]byte, index int) []ProofStep {
	if len(leaves) == 0 || index < 0 || index >= len(leaves) {
		return nil
	}
	var proof []ProofStep
	var level [][]byte
	for _, leaf := range leaves {
		level = append(level, hashLeaf(leaf))
	}

	currIndex := index

	for len(level) > 1 {
		var next [][]byte
		for i := 0; i < len(level); i += 2 {
			if i+1 < len(level) {
				next = append(next, hashNode(level[i], level[i+1]))
				//if our target node exists in this pair, we will add its neighbor to the proof.
				if i == currIndex {
					//target is on the left so we will add its right neighbor to the proof.
					proof = append(proof, ProofStep{Hash: level[i+1], IsLeft: false})
				} else if i+1 == currIndex {
					//target is on the right so we will add its left neighbor to the proof.
					proof = append(proof, ProofStep{Hash: level[i], IsLeft: true})
				}
			} else {
				next = append(next, level[i])
			}
		}
		level = next
		currIndex = currIndex / 2
	}
	return proof
}

func BuildLeaves(data map[string]string) [][]byte {
	var urls []string
	for u := range data {
		urls = append(urls, u)
	}

	sort.Strings(urls)

	var leaves [][]byte
	for _, u := range urls {
		hashStr := data[u]
		combined := []byte(u + hashStr)
		h := sha256.Sum256(combined)
		leaves = append(leaves, h[:])
	}
	return leaves
}
