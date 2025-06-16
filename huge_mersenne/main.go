package main

import (
	"fmt"
	"os"
)

//
// 2^p - 1
//

// GOAL: 333e6 < p < 1 billion (around 35 million primes)
// GOAL: On average 1 second per p

// Create a log file with all possible n's and their current processing step

//
// Finding easy factors
//

// Trial factoring
// Each factor q is of form 2kp + 1 and q must be 1 or 7 mod 8
// Modified sieve of eratosthenes
// Trial factoring with the powering algorithm from GIMPS

// P-1 method

//
// Finding a probably prime
//
// PRP

//
// Last needed test if probable prime
//
// Lucas-Lehmer test

func main() {
	fmt.Println("Hello, World!")

	os.Setenv("CGO_ENABLE", "1")
	fmt.Println(os.Getenv("CGO_ENABLE"))

	err := initDB()
	if err != nil {
		fmt.Printf("%v", err)
	}
}
