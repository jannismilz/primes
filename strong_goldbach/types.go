package main

import "time"

type Result struct {
	index            int
	recordCandidates []RecordCandidate
	maxTries         int
	maxTriesN        int
	totalTries       int64
	totalNumbers     int64
	averageTries     float64
	processedTime    time.Duration
	hash             string
}

type RecordCandidate struct {
	n    int
	minP int
}
