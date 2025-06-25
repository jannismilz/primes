package main

import (
	"fmt"
	"time"
)

func ProcessChunk(index int, start, end int, smallPrimes []int, recordTreshold int) Result {
	startTime := time.Now()
	var bufferSize int = 1e6 // Same as sieve_1e6
	var upperSieveStart int = end - bufferSize
	upperPrimesMap := SieveBetween(upperSieveStart, end)
	upperPrimes := make([]int, 0)

	if start%2 == 1 {
		start++
	}

	for i := 0; i < len(upperPrimesMap); i++ {
		if upperPrimesMap[i] {
			upperPrimes = append(upperPrimes, i+upperSieveStart)
		}
	}

	// Create a map for faster lookups
	isPrime := make(map[int]bool)
	for _, p := range smallPrimes {
		isPrime[p] = true
	}
	for _, p := range upperPrimes {
		isPrime[p] = true
	}

	var pairs = make([][2]int, 0, (end-start)/2+1)

	result := Result{
		index:            index,
		recordCandidates: make([]RecordCandidate, 0),
		totalTries:       0,
		totalNumbers:     0,
		hash:             "",
	}

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
		// TODO: Test further and if still not found we would have disproven the conjecture
		if !found {
			for _, p := range upperPrimes {
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
		if tries > recordTreshold {
			result.recordCandidates = append(result.recordCandidates, RecordCandidate{n, minPrime})
		}

		pairs = append(pairs, [2]int{n, minPrime})
	}

	result.processedTime = time.Since(startTime)
	result.averageTries = float64(result.totalTries) / float64(result.totalNumbers)
	result.hash = HashResults(pairs)

	fmt.Printf("Processed chunk starting at %d: max tries=%d for n=%d, avg=%.2f", start, result.maxTries, result.maxTriesN, result.averageTries)

	return result
}
