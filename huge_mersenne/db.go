package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"os"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFile    = "mersenne_candidates.db"
	startN    = 332192806 // 332_192_806
	endN      = 1000000000
	batchSize = 6000000 // For sieve operations
)

// createDBAndCreateTableIfNotExist initializes the SQLite database and tables
func createDBAndCreateTableIfNotExist() (*sql.DB, error) {
	// Check if database exists
	_, err := os.Stat(dbFile)
	dbExists := !os.IsNotExist(err)

	// Open database connection
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if !dbExists {
		log.Println("Creating new database...")

		// Create candidates table
		_, err = db.Exec(`CREATE TABLE candidates (
			p INTEGER PRIMARY KEY,
			trial_factored INTEGER,
			p1_factored INTEGER,
			prp_tested INTEGER,
			ll_tested INTEGER
		)`)
		if err != nil {
			return nil, fmt.Errorf("failed to create table: %v", err)
		}

		// Create indexes for performance
		_, err = db.Exec(`CREATE INDEX idx_trial ON candidates(trial_factored)`)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %v", err)
		}
		_, err = db.Exec(`CREATE INDEX idx_p1 ON candidates(p1_factored)`)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %v", err)
		}
		_, err = db.Exec(`CREATE INDEX idx_prp ON candidates(prp_tested)`)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %v", err)
		}
		_, err = db.Exec(`CREATE INDEX idx_ll ON candidates(ll_tested)`)
		if err != nil {
			return nil, fmt.Errorf("failed to create index: %v", err)
		}
	}

	// Set pragmas for performance
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to set journal mode: %v", err)
	}
	_, err = db.Exec("PRAGMA synchronous=NORMAL")
	if err != nil {
		return nil, fmt.Errorf("failed to set synchronous mode: %v", err)
	}

	return db, nil
}

// insertIntoDB inserts a new prime candidate into the database
func insertIntoDB(db *sql.DB, p int) error {
	_, err := db.Exec("INSERT INTO candidates (p) VALUES (?)", p)
	if err != nil {
		return fmt.Errorf("failed to insert candidate %d: %v", p, err)
	}
	return nil
}

// updateInDB updates a candidate's status after a method has been applied
func updateInDB(db *sql.DB, p int, method string, found bool) error {
	var column string
	switch method {
	case "trial":
		column = "trial_factored"
	case "p1":
		column = "p1_factored"
	case "prp":
		column = "prp_tested"
	case "ll":
		column = "ll_tested"
	default:
		return fmt.Errorf("unknown method: %s", method)
	}

	result := 0
	if found {
		result = 1
	}

	_, err := db.Exec(fmt.Sprintf("UPDATE candidates SET %s = ?, last_method = ? WHERE p = ?", column),
		result, method, p)
	if err != nil {
		return fmt.Errorf("failed to update candidate %d: %v", p, err)
	}
	return nil
}

// findNextWork gets the next candidate for a specific method
func findNextWork(db *sql.DB, method string) (int, error) {
	var column string
	switch method {
	case "trial":
		column = "trial_factored"
	case "p1":
		column = "p1_factored"
	case "prp":
		column = "prp_tested"
	case "ll":
		column = "ll_tested"
	default:
		return 0, fmt.Errorf("unknown method: %s", method)
	}

	var p int
	err := db.QueryRow(fmt.Sprintf("SELECT p FROM candidates WHERE %s IS NULL LIMIT 1", column)).Scan(&p)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("no candidates left for method %s", method)
		}
		return 0, fmt.Errorf("failed to get next candidate: %v", err)
	}
	return p, nil
}

// isPrime checks if a number is prime (simple trial division)
func isPrime(n int) bool {
	if n <= 1 {
		return false
	}
	if n <= 3 {
		return true
	}
	if n%2 == 0 || n%3 == 0 {
		return false
	}
	i := 5
	for i*i <= n {
		if n%i == 0 || n%(i+2) == 0 {
			return false
		}
		i += 6
	}
	return true
}

// Generate all primes from 332_192_806 to 1_000_000_000
func initDB() error {
	// Initialize database
	db, err := createDBAndCreateTableIfNotExist()
	if err != nil {
		return fmt.Errorf("failed to initialize database: %v", err)
	}
	defer db.Close()

	// Check if we need to populate the database
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM candidates").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check candidate count: %v", err)
	}

	if count == 0 {
		log.Printf("Populating prime candidates between %d and %d...\n", startN, endN)
		start := time.Now()

		// Process in batches to avoid memory issues
		for batchStart := startN; batchStart < endN; batchStart += batchSize {
			batchEnd := batchStart + batchSize
			if batchEnd > endN {
				batchEnd = endN
			}

			log.Printf("Processing batch %d to %d...\n", batchStart, batchEnd)

			// Create a sieve for this batch
			sieveSize := batchEnd - batchStart
			sieve := make([]bool, sieveSize)

			// Mark all as potential primes
			for i := range sieve {
				sieve[i] = true
			}

			// Apply sieve up to sqrt(batchEnd)
			limit := int(math.Sqrt(float64(batchEnd)))
			for i := 2; i <= limit; i++ {
				if isPrime(i) {
					// Find the first multiple of i in the batch
					start := ((batchStart + i - 1) / i) * i
					if start < batchStart {
						start += i
					}

					// Mark all multiples of i in the batch as non-prime
					for j := start; j < batchEnd; j += i {
						if j >= batchStart {
							sieve[j-batchStart] = false
						}
					}
				}
			}

			// Begin a transaction for batch insert
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %v", err)
			}

			// Insert primes from this batch
			count := 0

			// Collect candidates in batches for bulk insert
			const bulkSize = 10000
			valueStrings := make([]string, 0, bulkSize)
			valueArgs := make([]interface{}, 0, bulkSize)

			for i := 0; i < sieveSize; i++ {
				p := batchStart + i
				if sieve[i] && p > 1 {
					valueStrings = append(valueStrings, "(?)")
					valueArgs = append(valueArgs, p)
					count++

					// Execute bulk insert when batch is full or at the end
					if len(valueStrings) >= bulkSize || i == sieveSize-1 {
						if len(valueStrings) > 0 {
							stmt := fmt.Sprintf("INSERT INTO candidates (p) VALUES %s",
								strings.Join(valueStrings, ","))
							_, err = tx.Exec(stmt, valueArgs...)
							if err != nil {
								tx.Rollback()
								return fmt.Errorf("failed to bulk insert candidates: %v", err)
							}

							// Reset for next batch
							valueStrings = valueStrings[:0]
							valueArgs = valueArgs[:0]

							// Log progress periodically
							if count%100000 == 0 {
								log.Printf("Inserted %d candidates so far...", count)
							}
						}
					}
				}
			}

			err = tx.Commit()
			if err != nil {
				return fmt.Errorf("failed to commit transaction: %v", err)
			}

			log.Printf("Added %d prime candidates from batch\n", count)
		}

		elapsed := time.Since(start)
		log.Printf("Populated prime candidates in %s\n", elapsed)
	}

	// Print statistics
	printDBStats(db)

	return nil
}

// printDBStats prints statistics about the database
func printDBStats(db *sql.DB) {
	var total int
	db.QueryRow("SELECT COUNT(*) FROM candidates").Scan(&total)

	var trialPending, p1Pending, prpPending, llPending int
	db.QueryRow("SELECT COUNT(*) FROM candidates WHERE trial_factored IS NULL").Scan(&trialPending)
	db.QueryRow("SELECT COUNT(*) FROM candidates WHERE p1_factored IS NULL").Scan(&p1Pending)
	db.QueryRow("SELECT COUNT(*) FROM candidates WHERE prp_tested IS NULL").Scan(&prpPending)
	db.QueryRow("SELECT COUNT(*) FROM candidates WHERE ll_tested IS NULL").Scan(&llPending)

	var factorFound int
	db.QueryRow("SELECT COUNT(*) FROM candidates WHERE trial_factored = 1 OR p1_factored = 1").Scan(&factorFound)

	var mersenneFound int
	db.QueryRow("SELECT COUNT(*) FROM candidates WHERE ll_tested = 1").Scan(&mersenneFound)

	fmt.Printf("Total candidates: %d\n", total)
	fmt.Printf("Pending trial factoring: %d\n", trialPending)
	fmt.Printf("Pending P-1 factoring: %d\n", p1Pending)
	fmt.Printf("Pending PRP testing: %d\n", prpPending)
	fmt.Printf("Pending Lucas-Lehmer: %d\n", llPending)
	fmt.Printf("Factors found: %d\n", factorFound)
	fmt.Printf("Mersenne primes found: %d\n", mersenneFound)
}
