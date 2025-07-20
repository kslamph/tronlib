package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// QueryOptions represents options for querying event signatures
type QueryOptions struct {
	Limit     int
	EventName string
	Contract  string
	StartTime time.Time
	EndTime   time.Time
	SortBy    string // "last_seen", "usage_count", "first_seen"
	SortOrder string // "asc", "desc"
}

// parseTimeString parses a time string with flexible format support
func parseTimeString(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, nil
	}

	// Try different formats in order of preference
	formats := []string{
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02T15:04:05.999999-07:00",
		"2006-01-02T15:04:05.999999",
		"2006-01-02T15:04:05.999-07:00",
		"2006-01-02T15:04:05.999",
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05.999999-07:00",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05.999-07:00",
		"2006-01-02 15:04:05.999",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse time string: %s", timeStr)
}

// QueryEventSignatures queries event signatures with the given options
func (d *Database) QueryEventSignatures(opts QueryOptions) ([]EventSignature, error) {
	query := `
		SELECT id, signature, event_name, parameter_types, parameter_names, first_seen, last_seen, usage_count, contract_list
		FROM event_signatures
		WHERE 1=1
	`
	args := []interface{}{}

	if opts.EventName != "" {
		query += " AND event_name LIKE ?"
		args = append(args, "%"+opts.EventName+"%")
	}

	if opts.Contract != "" {
		query += " AND contract_list LIKE ?"
		args = append(args, "%"+opts.Contract+"%")
	}

	if !opts.StartTime.IsZero() {
		query += " AND first_seen >= ?"
		args = append(args, opts.StartTime)
	}

	if !opts.EndTime.IsZero() {
		query += " AND first_seen <= ?"
		args = append(args, opts.EndTime)
	}

	// Add sorting
	if opts.SortBy == "" {
		opts.SortBy = "last_seen"
	}
	if opts.SortOrder == "" {
		opts.SortOrder = "desc"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", opts.SortBy, opts.SortOrder)

	// Add limit
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query event signatures: %v", err)
	}
	defer rows.Close()

	var signatures []EventSignature
	for rows.Next() {
		var sig EventSignature
		var firstSeenStr, lastSeenStr string
		err := rows.Scan(&sig.ID, &sig.Signature, &sig.EventName, &sig.ParameterTypes, &sig.ParameterNames, &firstSeenStr, &lastSeenStr, &sig.UsageCount, &sig.ContractList)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event signature: %v", err)
		}

		// Parse time strings
		sig.FirstSeen, err = parseTimeString(firstSeenStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first_seen: %v", err)
		}
		sig.LastSeen, err = parseTimeString(lastSeenStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_seen: %v", err)
		}

		signatures = append(signatures, sig)
	}

	return signatures, nil
}

// GetStatistics returns statistics about collected event signatures
func (d *Database) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total signatures
	var totalSignatures int
	err := d.db.QueryRow("SELECT COUNT(*) FROM event_signatures").Scan(&totalSignatures)
	if err != nil {
		return nil, fmt.Errorf("failed to get total signatures: %v", err)
	}
	stats["total_signatures"] = totalSignatures

	// Known events (not unknown_event)
	var knownEvents int
	err = d.db.QueryRow("SELECT COUNT(*) FROM event_signatures WHERE event_name NOT LIKE 'unknown_event%'").Scan(&knownEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get known events: %v", err)
	}
	stats["known_events"] = knownEvents

	// Unknown events
	var unknownEvents int
	err = d.db.QueryRow("SELECT COUNT(*) FROM event_signatures WHERE event_name LIKE 'unknown_event%'").Scan(&unknownEvents)
	if err != nil {
		return nil, fmt.Errorf("failed to get unknown events: %v", err)
	}
	stats["unknown_events"] = unknownEvents

	// Total usage count
	var totalUsage sql.NullInt64
	err = d.db.QueryRow("SELECT SUM(usage_count) FROM event_signatures").Scan(&totalUsage)
	if err != nil {
		return nil, fmt.Errorf("failed to get total usage: %v", err)
	}
	if totalUsage.Valid {
		stats["total_usage"] = totalUsage.Int64
	} else {
		stats["total_usage"] = int64(0)
	}

	// Most used signature
	var mostUsedSignature, mostUsedEventName sql.NullString
	var mostUsedCount sql.NullInt64
	err = d.db.QueryRow(`
		SELECT signature, event_name, usage_count 
		FROM event_signatures 
		ORDER BY usage_count DESC 
		LIMIT 1
	`).Scan(&mostUsedSignature, &mostUsedEventName, &mostUsedCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get most used signature: %v", err)
	}

	if mostUsedSignature.Valid && mostUsedEventName.Valid && mostUsedCount.Valid {
		stats["most_used_signature"] = map[string]interface{}{
			"signature":   mostUsedSignature.String,
			"event_name":  mostUsedEventName.String,
			"usage_count": mostUsedCount.Int64,
		}
	} else {
		stats["most_used_signature"] = map[string]interface{}{
			"signature":   "",
			"event_name":  "",
			"usage_count": int64(0),
		}
	}

	// First and last seen
	var firstSeenStr, lastSeenStr sql.NullString
	err = d.db.QueryRow("SELECT MIN(first_seen), MAX(last_seen) FROM event_signatures").Scan(&firstSeenStr, &lastSeenStr)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get time range: %v", err)
	}

	var firstSeen, lastSeen time.Time
	if firstSeenStr.Valid {
		firstSeen, err = parseTimeString(firstSeenStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first_seen: %v", err)
		}
	}
	if lastSeenStr.Valid {
		lastSeen, err = parseTimeString(lastSeenStr.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse last_seen: %v", err)
		}
	}

	stats["first_seen"] = firstSeen
	stats["last_seen"] = lastSeen

	return stats, nil
}

// ExportToJSON exports event signatures to a JSON file
func (d *Database) ExportToJSON(filename string, opts QueryOptions) error {
	signatures, err := d.QueryEventSignatures(opts)
	if err != nil {
		return fmt.Errorf("failed to query signatures: %v", err)
	}

	// Convert to export format
	exportData := make([]map[string]interface{}, len(signatures))
	for i, sig := range signatures {
		var contractList []string
		if err := json.Unmarshal([]byte(sig.ContractList), &contractList); err != nil {
			contractList = []string{}
		}

		var parameterTypes []string
		if err := json.Unmarshal([]byte(sig.ParameterTypes), &parameterTypes); err != nil {
			parameterTypes = []string{}
		}

		var parameterNames []string
		if err := json.Unmarshal([]byte(sig.ParameterNames), &parameterNames); err != nil {
			parameterNames = []string{}
		}

		exportData[i] = map[string]interface{}{
			"signature":       sig.Signature,
			"event_name":      sig.EventName,
			"parameter_types": parameterTypes,
			"parameter_names": parameterNames,
			"contracts":       contractList,
			"first_seen":      sig.FirstSeen.Format(time.RFC3339),
			"last_seen":       sig.LastSeen.Format(time.RFC3339),
			"usage_count":     sig.UsageCount,
		}
	}

	// Write to file
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}

	log.Printf("Exported %d event signatures to %s", len(signatures), filename)
	return nil
}

// PrintEventSignatures prints event signatures in a formatted way
func PrintEventSignatures(signatures []EventSignature) {
	if len(signatures) == 0 {
		fmt.Println("No event signatures found.")
		return
	}

	fmt.Printf("\n=== Event Signatures (%d total) ===\n\n", len(signatures))

	for i, sig := range signatures {
		fmt.Printf("Signature #%d:\n", i+1)
		fmt.Printf("  Signature: 0x%s\n", sig.Signature)
		fmt.Printf("  Event: %s\n", sig.EventName)
		fmt.Printf("  Parameter Types: %s\n", sig.ParameterTypes)
		fmt.Printf("  Parameter Names: %s\n", sig.ParameterNames)
		fmt.Printf("  First Seen: %s\n", sig.FirstSeen.Format(time.RFC3339))
		fmt.Printf("  Last Seen: %s\n", sig.LastSeen.Format(time.RFC3339))
		fmt.Printf("  Usage Count: %d\n", sig.UsageCount)
		fmt.Printf("  Contracts: %s\n", sig.ContractList)
		fmt.Println()
	}
}

// PrintStatistics prints statistics in a formatted way
func PrintStatistics(stats map[string]interface{}) {
	fmt.Println("\n=== Event Signature Statistics ===")
	fmt.Printf("Total Signatures: %d\n", stats["total_signatures"])
	fmt.Printf("Known Events: %d\n", stats["known_events"])
	fmt.Printf("Unknown Events: %d\n", stats["unknown_events"])
	fmt.Printf("Total Usage: %d\n", stats["total_usage"])

	if mostUsed, ok := stats["most_used_signature"].(map[string]interface{}); ok {
		fmt.Printf("Most Used Signature: 0x%s (%s) - %d times\n",
			mostUsed["signature"], mostUsed["event_name"], mostUsed["usage_count"])
	}

	if firstSeen, ok := stats["first_seen"].(time.Time); ok {
		fmt.Printf("First Seen: %s\n", firstSeen.Format(time.RFC3339))
	}

	if lastSeen, ok := stats["last_seen"].(time.Time); ok {
		fmt.Printf("Last Seen: %s\n", lastSeen.Format(time.RFC3339))
	}
	fmt.Println()
}
