package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"veritect/internal/database"
	"veritect/internal/notifier"
)

const lockFile = "veritect.lock"

// lockData is the on-disk format for the schema baseline.
type lockData struct {
	Hash    string                `json:"hash"`
	Columns []database.ColumnInfo `json:"columns"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Fprintln(os.Stderr, "Error: DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	slackWebhook := os.Getenv("SLACK_WEBHOOK")

	// Fetch current schema
	columns, hash, err := database.FetchSchema(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching schema: %v\n", err)
		os.Exit(1)
	}

	// Check for existing lock file
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		// First run: initialize baseline
		data := lockData{Hash: hash, Columns: columns}
		b, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshaling lock file: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(lockFile, b, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing lock file: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("veritect.lock initialized with current schema baseline.")
		os.Exit(0)
	}

	// Read stored baseline
	b, err := os.ReadFile(lockFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading lock file: %v\n", err)
		os.Exit(1)
	}

	var stored lockData
	if err := json.Unmarshal(b, &stored); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing lock file: %v\n", err)
		os.Exit(1)
	}

	// Compare hashes
	if hash == stored.Hash {
		fmt.Println("Schema unchanged. No drift detected.")
		os.Exit(0)
	}

	// Drift detected
	added, removed, modified := database.Diff(stored.Columns, columns)

	fmt.Fprintln(os.Stderr, "ERROR: Schema drift detected!")
	fmt.Fprintf(os.Stderr, "Added: %d | Removed: %d | Modified: %d\n", len(added), len(removed), len(modified))

	if len(added) > 0 {
		fmt.Fprintln(os.Stderr, "\nAdded columns:")
		for _, c := range added {
			fmt.Fprintf(os.Stderr, "  + %s.%s (%s, nullable: %s)\n", c.TableName, c.ColumnName, c.DataType, c.IsNullable)
		}
	}
	if len(removed) > 0 {
		fmt.Fprintln(os.Stderr, "\nRemoved columns:")
		for _, c := range removed {
			fmt.Fprintf(os.Stderr, "  - %s.%s (%s, nullable: %s)\n", c.TableName, c.ColumnName, c.DataType, c.IsNullable)
		}
	}
	if len(modified) > 0 {
		fmt.Fprintln(os.Stderr, "\nModified columns:")
		for _, c := range modified {
			fmt.Fprintf(os.Stderr, "  ~ %s.%s (%s, nullable: %s)\n", c.TableName, c.ColumnName, c.DataType, c.IsNullable)
		}
	}

	// Notify Slack if configured
	if slackWebhook != "" {
		if err := notifier.SendDiff(slackWebhook, added, removed, modified); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to send Slack notification: %v\n", err)
		}
	}

	os.Exit(1)
}
