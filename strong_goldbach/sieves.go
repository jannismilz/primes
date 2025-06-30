package main

import (
	"math"
)

func SimpleSieve() []int {
	primes := make([]bool, 20000)
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

func SieveForChunk(start int, size int) map[int]bool {
	// Using a map where keys are the prime numbers
	primeMap := make(map[int]bool)

	// Initialize all numbers from 2 to size-1 as potential primes
	for i := 2; i < size; i++ {
		primeMap[start+i] = true
	}

	// Apply the sieve
	for i := 2; i < size; i++ {
		if primeMap[i] {
			for j := i * i; j < size; j += i {
				delete(primeMap, j)
			}
		}
	}

	return primeMap
}

func SieveBetween(start int, end int) []bool {
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
