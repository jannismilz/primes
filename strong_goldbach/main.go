package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	. "github.com/klauspost/cpuid/v2"
	"os"
	"runtime/pprof"
	"sync"
	"time"
)

// Choose range of 10^10 numbers
// Sieve primes up to 50'000

// Create global variables for average tries per n (even number)
// Track n with the highest tries so far

// Process in chunks of 10^7
// Create sieve from chunk * 10^7-50'000 to chunk * 10^7
// Check all even numbers in the range and search for min_p so that n - min_p = is prime

// Store all n where min_p is larger than 8419
// Currently on work laptop: 16m3s

var RANGE_START int = 1e8 // later: 10_000_000_000
var RANGE_END int = 2e8   // later: 20_000_000_000
var CHUNK_SIZE int = 1e6
var RECORD_TRESHOLD int = 8419

func main() {
	f, err := os.Create("cpu.prof")
	if err != nil {
		panic(err)
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()

	fmt.Println("Verifying Strong Goldbach Conjecture")
	fmt.Printf("Range: %d to %d\n", RANGE_START, RANGE_END)

	startTime := time.Now()

	// Get all primes up to 1e6 for initial checking
	smallPrimes := SimpleSieve()
	fmt.Printf("Generated %d small primes up to 1,000,000\n", len(smallPrimes))

	// Get all chunks to process
	chunks := GetChunks(RANGE_START, RANGE_END, CHUNK_SIZE)
	fmt.Printf("Processing %d chunks of size %d\n", len(chunks), CHUNK_SIZE)

	// Process chunks in parallel
	results := processChunks(chunks, smallPrimes)

	// Aggregate results
	maxTries := 0
	maxTriesN := 0
	var totalTries int64 = 0
	var totalNumbers int64 = 0
	var totalTime time.Duration = 0
	var totalRecordCandidates int = 0
	var maxRecordCandidate RecordCandidate

	// Sort chunks by their start value to ensure consistent hashing order
	sortedResults := make([]Result, len(chunks))

	// Place results in the correct order based on index
	for _, result := range results {
		sortedResults[result.index] = result
	}

	// Create a merkle tree of hashes
	levelHashes := make([]string, len(sortedResults))
	for i, result := range sortedResults {
		levelHashes[i] = result.hash

		// Update statistics
		if result.maxTries > maxTries {
			maxTries = result.maxTries
			maxTriesN = result.maxTriesN
		} else if result.maxTries == maxTries {
			if result.maxTriesN < maxTriesN {
				maxTriesN = result.maxTriesN
			}
		}

		totalTries += result.totalTries
		totalNumbers += result.totalNumbers
		totalTime += result.processedTime

		// Count record candidates and find max
		totalRecordCandidates += len(result.recordCandidates)

		// Check for max record candidate
		for _, candidate := range result.recordCandidates {
			if candidate.n > maxRecordCandidate.n {
				maxRecordCandidate = candidate
			}
		}
	}

	// Build the merkle tree
	finalHash := buildMerkleRoot(levelHashes)
	averageTries := float64(totalTries) / float64(totalNumbers)

	fmt.Printf("\nResults:\n")
	fmt.Printf("Total even numbers checked: %d\n", totalNumbers)
	fmt.Printf("Total tries: %d\n", totalTries)
	fmt.Printf("Average tries per number: %.2f\n", averageTries)
	fmt.Printf("Maximum tries: %d (for n=%d)\n", maxTries, maxTriesN)
	fmt.Printf("Total record candidates: %d\n", totalRecordCandidates)
	if maxRecordCandidate.n > 0 {
		fmt.Printf("Max record candidate: n=%d, minPrime=%d\n", maxRecordCandidate.n, maxRecordCandidate.minP)
	}
	fmt.Printf("Total processing time: %.4fs\n", totalTime.Seconds())
	fmt.Printf("Total elapsed time: %.4fs\n", time.Since(startTime).Seconds())
	fmt.Printf("Verification hash: %s\n", finalHash)
	fmt.Printf("CPU Name: %s\n", CPU.BrandName)
	fmt.Printf("CPU Frequency: %d\n", CPU.Hz)
	fmt.Println("CPU Cores:", CPU.PhysicalCores)
}

func processChunks(chunks []int, smallPrimes []int) []Result {
	numWorkers := CPU.PhysicalCores
	var wg sync.WaitGroup
	resultChan := make(chan Result, len(chunks))
	chunkChan := make(chan int, len(chunks))

	// Send all chunks to the channel with their index
	for _, chunk := range chunks {
		chunkChan <- chunk
	}
	close(chunkChan)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for chunkStart := range chunkChan {
				// Find the index of this chunk
				index := -1
				for i, c := range chunks {
					if c == chunkStart {
						index = i
						break
					}
				}

				if index == -1 {
					fmt.Printf("Error: Could not find index for chunk %d\n", chunkStart)
					continue
				}

				result := ProcessChunk(index, chunkStart, chunkStart+CHUNK_SIZE-1, smallPrimes, RECORD_TRESHOLD)
				resultChan <- result
				fmt.Printf("Processed chunk %d starting at %d: candidates=%d, avg=%.2f\n",
					index, chunkStart, len(result.recordCandidates), result.averageTries)
			}
		}()
	}

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	// Collect results
	results := make([]Result, 0, len(chunks))
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// Function to build a merkle root hash from a list of hashes
func buildMerkleRoot(hashes []string) string {
	if len(hashes) == 0 {
		return ""
	}

	if len(hashes) == 1 {
		return hashes[0]
	}

	// Concatenate all hashes together
	allHashes := ""
	for _, hash := range hashes {
		allHashes += hash
	}

	// Hash the concatenated string
	h := sha256.New()
	h.Write([]byte(allHashes))
	return hex.EncodeToString(h.Sum(nil))
}
