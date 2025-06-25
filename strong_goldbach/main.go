package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	. "github.com/klauspost/cpuid/v2"
	"math"
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

var RANGE_START int = 1e9 // later: 10_000_000_000
var RANGE_END int = 2e9   // later: 20_000_000_000
var CHUNK_SIZE int = 1e7

type Result struct {
	maxTries      int
	maxTriesN     int
	totalTries    int64
	totalNumbers  int64
	averageTries  float64
	processedTime time.Duration
	pairs         [][2]int // Store (n, minPrime) pairs for verification
}

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

	// Get all primes up to 50k for initial checking
	smallPrimes := sieve_50k()
	fmt.Printf("Generated %d small primes up to 50,000\n", len(smallPrimes))

	// Get all chunks to process
	chunks := get_chunks()
	fmt.Printf("Processing %d chunks of size %d\n", len(chunks), CHUNK_SIZE)

	// Process chunks in parallel
	results := processChunks(chunks, smallPrimes)

	// Aggregate results
	maxTries := 0
	maxTriesN := 0
	var totalTries int64 = 0
	var totalNumbers int64 = 0
	var totalTime time.Duration = 0

	// Create a global hasher for all pairs
	globalHasher := sha256.New()

	// Sort chunks by their start value to ensure consistent hashing order
	sortedChunks := make([]int, len(chunks))
	copy(sortedChunks, chunks)

	// Process results in order of chunks
	for _, chunkStart := range sortedChunks {
		// Find the result for this chunk
		var result Result
		for _, r := range results {
			if r.pairs != nil && len(r.pairs) > 0 && r.pairs[0][0] >= chunkStart && r.pairs[0][0] < chunkStart+CHUNK_SIZE {
				result = r
				break
			}
		}

		if result.pairs == nil {
			continue
		}

		// Update statistics
		if result.maxTries > maxTries {
			maxTries = result.maxTries
			maxTriesN = result.maxTriesN
		}
		totalTries += result.totalTries
		totalNumbers += result.totalNumbers
		totalTime += result.processedTime

		// Add all pairs from this chunk to the global hash
		for _, pair := range result.pairs {
			n := pair[0]
			minPrime := pair[1]

			// Hash the (n, minPrime) pair
			buf := make([]byte, 16)
			binary.BigEndian.PutUint64(buf[:8], uint64(n))
			binary.BigEndian.PutUint64(buf[8:], uint64(minPrime))
			globalHasher.Write(buf)
		}
	}

	// Generate the final hash
	finalHash := hex.EncodeToString(globalHasher.Sum(nil))
	averageTries := float64(totalTries) / float64(totalNumbers)

	fmt.Printf("\nResults:\n")
	fmt.Printf("Total even numbers checked: %d\n", totalNumbers)
	fmt.Printf("Total tries: %d\n", totalTries)
	fmt.Printf("Average tries per number: %.2f\n", averageTries)
	fmt.Printf("Maximum tries: %d (for n=%d)\n", maxTries, maxTriesN)
	fmt.Printf("Total processing time: %.4fs\n", totalTime.Seconds())
	fmt.Printf("Total elapsed time: %.4fs\n", time.Since(startTime).Seconds())
	fmt.Printf("Verification hash: %s\n", finalHash)
	fmt.Printf("CPU Name: %s\n", CPU.BrandName)
	fmt.Printf("CPU Frequency: %d\n", CPU.Hz)
	fmt.Println("CPU Cores:", CPU.PhysicalCores)
}

func sieve_50k() []int {
	primes := make([]bool, 50_001)
	var result []int

	for i := range primes {
		primes[i] = true
	}

	primes[0] = false
	primes[1] = false

	for i, isPrime := range primes {
		if !isPrime {
			continue
		}
		result = append(result, i)
		for j := i * i; j < len(primes); j += i {
			primes[j] = false
		}
	}

	return result
}

func sieve_between(start, end int) []bool {
	size := end - start + 1
	primes := make([]bool, size)

	// Initialize all as prime
	for i := range primes {
		primes[i] = true
	}

	// Handle special cases for 0 and 1
	if start <= 0 && end >= 0 {
		primes[0-start] = false
	}
	if start <= 1 && end >= 1 {
		primes[1-start] = false
	}

	// Apply sieve
	sqrtEnd := int(math.Sqrt(float64(end)))
	for i := 2; i <= sqrtEnd; i++ {
		// Find the first multiple of i in the range
		firstMultiple := start
		if firstMultiple < i*i {
			firstMultiple = i * i
		} else {
			firstMultiple = ((start + i - 1) / i) * i // Round up to nearest multiple of i
		}

		// Mark all multiples as non-prime
		for j := firstMultiple; j <= end; j += i {
			primes[j-start] = false
		}
	}

	return primes
}

func get_chunks() []int {
	chunks := make([]int, 0)

	for i := RANGE_START; i < RANGE_END; i += CHUNK_SIZE {
		chunks = append(chunks, i)
	}

	return chunks
}

func processChunks(chunks []int, smallPrimes []int) []Result {
	numWorkers := CPU.PhysicalCores
	var wg sync.WaitGroup
	resultChan := make(chan Result, len(chunks))
	chunkChan := make(chan int, len(chunks))

	// Send all chunks to the channel
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
				result := processChunk(chunkStart, chunkStart+CHUNK_SIZE-1, smallPrimes)
				resultChan <- result
				fmt.Printf("Processed chunk starting at %d: max tries=%d for n=%d, avg=%.2f, pairs=%d\n",
					chunkStart, result.maxTries, result.maxTriesN, result.averageTries, len(result.pairs))
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

func processChunk(start, end int, smallPrimes []int) Result {
	startTime := time.Now()

	// Ensure we only process even numbers
	if start%2 == 1 {
		start++
	}

	// Generate primes for this chunk and a bit before (for checking)
	bufferSize := 50_000 // Buffer to ensure we have enough primes for checking
	chunkPrimesMap := sieve_between(start-bufferSize, end)

	// Convert boolean map to slice of actual prime numbers
	chunkPrimes := make([]int, 0)
	for i := 0; i < len(chunkPrimesMap); i++ {
		if chunkPrimesMap[i] {
			chunkPrimes = append(chunkPrimes, i+(start-bufferSize))
		}
	}

	// Create a map for faster lookups
	isPrime := make(map[int]bool)
	for _, p := range smallPrimes {
		isPrime[p] = true
	}
	for _, p := range chunkPrimes {
		isPrime[p] = true
	}

	result := Result{
		maxTries:     0,
		maxTriesN:    0,
		totalTries:   0,
		totalNumbers: 0,
		pairs:        make([][2]int, 0, (end-start)/2+1), // Pre-allocate for all even numbers
	}

	// Check each even number in the range
	for n := start; n <= end; n += 2 {
		if n <= 2 {
			continue // Skip 2 as it's not applicable to Goldbach conjecture
		}

		result.totalNumbers++
		tries := 0
		found := false
		var minPrime int

		// Try each prime p and check if n-p is also prime
		for _, p := range smallPrimes {
			if p >= n {
				break
			}

			tries++
			complement := n - p

			if isPrime[complement] {
				found = true
				minPrime = p
				break
			}
		}

		// If not found in small primes, check in chunk primes
		if !found {
			for _, p := range chunkPrimes {
				if p >= n {
					break
				}

				tries++
				complement := n - p

				if isPrime[complement] {
					found = true
					minPrime = p
					break
				}
			}
		}

		// Update statistics
		result.totalTries += int64(tries)
		if tries > result.maxTries {
			result.maxTries = tries
			result.maxTriesN = n
		}

		// Verify the conjecture
		if !found {
			fmt.Printf("CONJECTURE FAILED for n=%d\n", n)
		} else {
			// Store the (n, minPrime) pair for later hashing
			result.pairs = append(result.pairs, [2]int{n, minPrime})
		}
	}

	result.processedTime = time.Since(startTime)
	result.averageTries = float64(result.totalTries) / float64(result.totalNumbers)

	return result
}
