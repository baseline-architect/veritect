package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/jackc/pgx/v5"
)

// ColumnInfo represents a single column's structural metadata.
type ColumnInfo struct {
	TableName            string `json:"table_name"`
	ColumnName           string `json:"column_name"`
	DataType             string `json:"data_type"`
	IsNullable           string `json:"is_nullable"`
	CharacterMaximumLength *int   `json:"character_maximum_length,omitempty"`
}

// FetchSchema connects to the given PostgreSQL database, pulls column metadata
// from the public schema, and returns both the ordered slice of columns and a
// deterministic SHA-256 hash of that structure.
func FetchSchema(ctx context.Context, connStr string) ([]ColumnInfo, string, error) {
	conn, err := pgx.Connect(ctx, connStr)
	if err != nil {
		return nil, "", fmt.Errorf("connecting to database: %w", err)
	}
	defer conn.Close(ctx)

	rows, err := conn.Query(ctx, `
		SELECT
			table_name,
			column_name,
			data_type,
			is_nullable,
			character_maximum_length
		FROM information_schema.columns
		WHERE table_schema = 'public'
		ORDER BY table_name ASC, column_name ASC
	`)
	if err != nil {
		return nil, "", fmt.Errorf("querying information_schema.columns: %w", err)
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		var charMaxLen *int
		if err := rows.Scan(
			&col.TableName,
			&col.ColumnName,
			&col.DataType,
			&col.IsNullable,
			&charMaxLen,
		); err != nil {
			return nil, "", fmt.Errorf("scanning column row: %w", err)
		}
		col.CharacterMaximumLength = charMaxLen
		columns = append(columns, col)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("iterating column rows: %w", err)
	}

	hash, err := hashColumns(columns)
	if err != nil {
		return nil, "", fmt.Errorf("hashing columns: %w", err)
	}

	return columns, hash, nil
}

// hashColumns produces a deterministic SHA-256 hex digest of the given columns.
func hashColumns(columns []ColumnInfo) (string, error) {
	// Re-serializing through JSON guarantees a stable, canonical representation.
	b, err := json.Marshal(columns)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// Diff compares two column snapshots and returns added, removed, and modified columns.
func Diff(oldCols, newCols []ColumnInfo) (added, removed, modified []ColumnInfo) {
	oldMap := make(map[string]ColumnInfo, len(oldCols))
	for _, c := range oldCols {
		key := c.TableName + "." + c.ColumnName
		oldMap[key] = c
	}

	newMap := make(map[string]ColumnInfo, len(newCols))
	for _, c := range newCols {
		key := c.TableName + "." + c.ColumnName
		newMap[key] = c
	}

	// Added: present in new, absent in old
	for key, col := range newMap {
		if _, ok := oldMap[key]; !ok {
			added = append(added, col)
		}
	}

	// Removed: present in old, absent in new
	for key, col := range oldMap {
		if _, ok := newMap[key]; !ok {
			removed = append(removed, col)
		}
	}

	// Modified: same key, different struct value
	for key, newCol := range newMap {
		if oldCol, ok := oldMap[key]; ok {
			if !columnsEqual(oldCol, newCol) {
				modified = append(modified, newCol)
			}
		}
	}

	sort.Slice(added, func(i, j int) bool {
		if added[i].TableName != added[j].TableName {
			return added[i].TableName < added[j].TableName
		}
		return added[i].ColumnName < added[j].ColumnName
	})
	sort.Slice(removed, func(i, j int) bool {
		if removed[i].TableName != removed[j].TableName {
			return removed[i].TableName < removed[j].TableName
		}
		return removed[i].ColumnName < removed[j].ColumnName
	})
	sort.Slice(modified, func(i, j int) bool {
		if modified[i].TableName != modified[j].TableName {
			return modified[i].TableName < modified[j].TableName
		}
		return modified[i].ColumnName < modified[j].ColumnName
	})

	return added, removed, modified
}

func columnsEqual(a, b ColumnInfo) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}
