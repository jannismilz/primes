package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

func GetChunks(start int, end int, chunk_size int) []int {
	chunks := make([]int, (end-start)/chunk_size)

	for i := start; i < end; i += chunk_size {
		chunks[(i-start)/chunk_size] = i
	}

	return chunks
}

func HashResults(pairs [][2]int) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", pairs)))

	return hex.EncodeToString(h.Sum(nil))
}
