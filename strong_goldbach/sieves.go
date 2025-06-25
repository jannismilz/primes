package main

import "math"

func SimpleSieve() []int {
	primes := make([]bool, 1e6)
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
