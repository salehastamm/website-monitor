package merkle

import (
	"bytes"
	"crypto/sha256"
	"testing"
)

func generateLeaves(n int) [][]byte {
	var leaves [][]byte
	for i := 0; i < n; i++ {
		leaves = append(leaves, []byte{byte(i)})
	}
	return leaves
}

func TestProofAndVerify_RoundTrip(t *testing.T) {
	//leaf quantity, edge cases
	sizes := []int{1, 3, 4, 8}

	for _, size := range sizes {
		leaves := generateLeaves(size)
		root := Root(leaves)

		for i := 0; i < size; i++ {
			proof := Proof(leaves, i)
			isValid := Verify(leaves[i], proof, root)

			if !isValid {
				t.Errorf("Verify failed! Tree size: %d, Index: %d", size, i)
			}
		}
	}
}

func TestVerify_TamperedLeaf(t *testing.T) {
	leaves := generateLeaves(4)
	root := Root(leaves)
	proof := Proof(leaves, 0)

	tamperedLeaf := []byte("fake data")

	isValid := Verify(tamperedLeaf, proof, root)
	if isValid {
		t.Errorf("Expected Verify to fail with a tampered leaf, but it succeeded")
	}
}

func TestVerify_TamperedProof(t *testing.T) {
	leaves := generateLeaves(4)
	root := Root(leaves)
	proof := Proof(leaves, 0)

	if len(proof) > 0 {
		// flip the first byte by XORing it with 0xFF
		proof[0].Hash[0] ^= 0xFF

		isValid := Verify(leaves[0], proof, root)
		if isValid {
			t.Errorf("Expected Verify to fail with a tampered proof, but it succeeded")
		}
	}
}

func TestDeterminism(t *testing.T) {
	data := map[string]string{
		"https://example.com":     "hashA",
		"https://google.com":      "hashB",
		"https://github.com":      "hashC",
		"https://invalid.example": "hashD",
		"https://golang.org":      "hashE",
	}
	leavesReference := BuildLeaves(data)
	rootReference := Root(leavesReference)

	for i := 0; i < 10; i++ {
		leaves := BuildLeaves(data)
		root := Root(leaves)

		if !bytes.Equal(rootReference, root) {
			t.Errorf("Determinism failed! Expected root %x, got %x on iteration %d", rootReference, root, i)
		}
	}
}

func TestRootEmpty(t *testing.T) {
	if root := Root(nil); root != nil {
		t.Errorf("Root(nil) should be nil, got %x", root)
	}
	if root := Root([][]byte{}); root != nil {
		t.Errorf("Root(empty slice) should be nil, got %x", root)
	}
}

func TestProofInvalidIndex(t *testing.T) {
	leaves := generateLeaves(4)

	for _, idx := range []int{-1, 4, 99} {
		if proof := Proof(leaves, idx); proof != nil {
			t.Errorf("Proof(leaves, %d) should be nil, got %d steps", idx, len(proof))
		}
	}

	if proof := Proof(nil, 0); proof != nil {
		t.Errorf("Proof(nil, 0) should be nil, got %d steps", len(proof))
	}
}

func TestBuildLeavesSortsByURL(t *testing.T) {
	data := map[string]string{
		"https://google.com":      "hashB",
		"https://example.com":     "hashA",
		"https://invalid.example": "hashD",
		"https://github.com":      "hashC",
		"https://golang.org":      "hashE",
	}

	// ترتیب الفبایی URLها، بدون توجه به ترتیب درج در map بالا
	wantOrder := []string{
		"https://example.com" + "hashA",
		"https://github.com" + "hashC",
		"https://golang.org" + "hashE",
		"https://google.com" + "hashB",
		"https://invalid.example" + "hashD",
	}

	leaves := BuildLeaves(data)

	if len(leaves) != len(wantOrder) {
		t.Fatalf("expected %d leaves, got %d", len(wantOrder), len(leaves))
	}

	for i, want := range wantOrder {
		expected := sha256.Sum256([]byte(want))
		if !bytes.Equal(leaves[i], expected[:]) {
			t.Errorf("leaf %d is out of order: expected hash of %q", i, want)
		}
	}
}
