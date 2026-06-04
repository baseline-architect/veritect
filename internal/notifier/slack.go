package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"veritect/internal/database"
)

// SendDiff delivers a structured Slack message containing the schema drift summary
// and detailed structural changes to the provided webhook URL.
func SendDiff(webhookURL string, added, removed, modified []database.ColumnInfo) error {
	if _, err := url.ParseRequestURI(webhookURL); err != nil {
		return fmt.Errorf("invalid slack webhook URL: %w", err)
	}

	payload := buildPayload(added, removed, modified)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling slack payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating slack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sending slack request: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func buildPayload(added, removed, modified []database.ColumnInfo) map[string]interface{} {
	var blocks []map[string]interface{}

	// Header
	blocks = append(blocks, map[string]interface{}{
		"type": "header",
		"text": map[string]interface{}{
			"type": "plain_text",
			"text": "Veritect: Schema Drift Detected",
		},
	})

	// Summary
	summary := fmt.Sprintf(
		"Added: %d | Removed: %d | Modified: %d",
		len(added), len(removed), len(modified),
	)
	blocks = append(blocks, map[string]interface{}{
		"type": "section",
		"text": map[string]interface{}{
			"type": "mrkdwn",
			"text": "*" + summary + "*",
		},
	})

	// Divider
	blocks = append(blocks, map[string]interface{}{"type": "divider"})

	// Added
	if len(added) > 0 {
		blocks = append(blocks, sectionBlock("Added Columns", formatColumns(added)))
	}

	// Removed
	if len(removed) > 0 {
		blocks = append(blocks, sectionBlock("Removed Columns", formatColumns(removed)))
	}

	// Modified
	if len(modified) > 0 {
		blocks = append(blocks, sectionBlock("Modified Columns", formatColumns(modified)))
	}

	return map[string]interface{}{
		"blocks": blocks,
	}
}

func sectionBlock(title, text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "section",
		"text": map[string]interface{}{
			"type": "mrkdwn",
			"text": "*" + title + "*\n" + text,
		},
	}
}

func formatColumns(cols []database.ColumnInfo) string {
	var out string
	for _, c := range cols {
		line := fmt.Sprintf(
			"• `%s.%s` — %s (nullable: %s",
			c.TableName, c.ColumnName, c.DataType, c.IsNullable,
		)
		if c.CharacterMaximumLength != nil {
			line += fmt.Sprintf(", max_length: %d", *c.CharacterMaximumLength)
		}
		line += ")\n"
		out += line
	}
	return out
}
