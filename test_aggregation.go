package main

import (
	"fmt"
	"time"
)

func main() {
	// Entry timestamp from chat history
	entryTimestamp := int64(1778087875739) // milliseconds
	
	// Convert to time
	entryTime := time.Unix(entryTimestamp/1000, (entryTimestamp%1000)*1000000)
	
	// Today's date range
	now := time.Now()
	today := now.Format("2006-01-02")
	start, _ := time.Parse("2006-01-02", today)
	end := start.Add(24 * time.Hour)
	
	fmt.Println("Entry timestamp (ms):", entryTimestamp)
	fmt.Println("Entry time:", entryTime)
	fmt.Println("Entry date:", entryTime.Format("2006-01-02"))
	fmt.Println()
	fmt.Println("Now:", now)
	fmt.Println("Today:", today)
	fmt.Println("Start:", start)
	fmt.Println("End:", end)
	fmt.Println()
	fmt.Println("Entry before start?", entryTime.Before(start))
	fmt.Println("Entry after end?", entryTime.After(end))
	fmt.Println("Entry in range?", !entryTime.Before(start) && !entryTime.After(end))
}
