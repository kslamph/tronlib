package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"event_signature_collector/database"
)

func main() {
	// Define command line flags
	var (
		dbPath    = flag.String("db", "event_signatures.db", "Path to the SQLite database")
		limit     = flag.Int("limit", 0, "Limit number of results (0 = no limit)")
		eventName = flag.String("event", "", "Filter by event name (partial match)")
		contract  = flag.String("contract", "", "Filter by contract address (partial match)")
		sortBy    = flag.String("sort", "last_seen", "Sort by: last_seen, usage_count, first_seen")
		sortOrder = flag.String("order", "desc", "Sort order: asc, desc")
		stats     = flag.Bool("stats", false, "Show statistics instead of signatures")
		export    = flag.String("export", "", "Export to JSON file")
		startTime = flag.String("start", "", "Filter by start time (RFC3339 format)")
		endTime   = flag.String("end", "", "Filter by end time (RFC3339 format)")
		cleanup   = flag.Bool("cleanup", false, "Clean up unknown events from database")
	)
	flag.Parse()

	// Initialize database
	db, err := database.NewDatabase(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Parse time filters
	var start, end time.Time
	if *startTime != "" {
		start, err = time.Parse(time.RFC3339, *startTime)
		if err != nil {
			log.Fatalf("Invalid start time format: %v", err)
		}
	}
	if *endTime != "" {
		end, err = time.Parse(time.RFC3339, *endTime)
		if err != nil {
			log.Fatalf("Invalid end time format: %v", err)
		}
	}

	// Build query options
	opts := database.QueryOptions{
		Limit:     *limit,
		EventName: *eventName,
		Contract:  *contract,
		StartTime: start,
		EndTime:   end,
		SortBy:    *sortBy,
		SortOrder: *sortOrder,
	}

	// Clean up unknown events
	if *cleanup {
		if err := db.CleanupUnknownEvents(); err != nil {
			log.Fatalf("Failed to cleanup unknown events: %v", err)
		}
		fmt.Println("Successfully cleaned up unknown events from database")
		return
	}

	// Show statistics
	if *stats {
		stats, err := db.GetStatistics()
		if err != nil {
			log.Fatalf("Failed to get statistics: %v", err)
		}
		database.PrintStatistics(stats)
		return
	}

	// Export to JSON
	if *export != "" {
		if err := db.ExportToJSON(*export, opts); err != nil {
			log.Fatalf("Failed to export to JSON: %v", err)
		}
		fmt.Printf("Exported data to %s\n", *export)
		return
	}

	// Query and display signatures
	signatures, err := db.QueryEventSignatures(opts)
	if err != nil {
		log.Fatalf("Failed to query signatures: %v", err)
	}

	database.PrintEventSignatures(signatures)
}

func init() {
	// Set up usage information
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Event Signature Query Tool\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -limit 10                    # Show 10 most recent signatures\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -event Transfer              # Show Transfer events\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -contract TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t  # Show events from USDT contract\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -sort usage_count -order desc -limit 5  # Show 5 most used signatures\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -stats                       # Show statistics\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -cleanup                     # Clean up unknown events\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -export signatures.json      # Export all signatures to JSON\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -start 2024-01-01T00:00:00Z -end 2024-01-02T00:00:00Z  # Filter by date range\n", os.Args[0])
	}
}
