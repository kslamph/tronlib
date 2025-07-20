package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Database represents the SQLite database for storing event signatures
type Database struct {
	db *sql.DB
}

// EventSignature represents a unique event signature with its parameters
type EventSignature struct {
	ID             int64
	Signature      string // Event signature hash (32 bytes hex)
	EventName      string // Decoded event name
	ParameterTypes string // JSON array of parameter types
	ParameterNames string // JSON array of parameter names
	FirstSeen      time.Time
	LastSeen       time.Time
	UsageCount     int64
	ContractList   string // JSON array of contract addresses (up to 10)
}

// ContractUsage represents contract usage of an event signature
type ContractUsage struct {
	ID           int64
	SignatureID  int64
	ContractAddr string
	FirstSeen    time.Time
	LastSeen     time.Time
	UsageCount   int64
}

// NewDatabase creates a new database connection and initializes tables
func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// Initialize tables
	if err := initTables(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize tables: %v", err)
	}

	database := &Database{db: db}

	// Migrate old database structure if needed
	if err := database.MigrateOldDatabase(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %v", err)
	}

	return database, nil
}

// Close closes the database connection
func (d *Database) Close() error {
	return d.db.Close()
}

// initTables creates the necessary tables if they don't exist
func initTables(db *sql.DB) error {
	// Event signatures table
	eventSignaturesSQL := `
	CREATE TABLE IF NOT EXISTS event_signatures (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		signature TEXT UNIQUE NOT NULL,
		event_name TEXT NOT NULL,
		parameter_types TEXT NOT NULL,
		parameter_names TEXT NOT NULL,
		first_seen DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		usage_count INTEGER DEFAULT 1,
		contract_list TEXT NOT NULL
	);`

	// Contract usage table
	contractUsageSQL := `
	CREATE TABLE IF NOT EXISTS contract_usage (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		signature_id INTEGER NOT NULL,
		contract_addr TEXT NOT NULL,
		first_seen DATETIME NOT NULL,
		last_seen DATETIME NOT NULL,
		usage_count INTEGER DEFAULT 1,
		FOREIGN KEY (signature_id) REFERENCES event_signatures (id),
		UNIQUE(signature_id, contract_addr)
	);`

	// Create indexes for better performance
	indexesSQL := `
	CREATE INDEX IF NOT EXISTS idx_signature ON event_signatures (signature);
	CREATE INDEX IF NOT EXISTS idx_event_name ON event_signatures (event_name);
	CREATE INDEX IF NOT EXISTS idx_contract_usage_signature ON contract_usage (signature_id);
	CREATE INDEX IF NOT EXISTS idx_contract_usage_contract ON contract_usage (contract_addr);
	`

	queries := []string{eventSignaturesSQL, contractUsageSQL, indexesSQL}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %v", err)
		}
	}

	return nil
}

// SaveEventSignature saves or updates an event signature
func (d *Database) SaveEventSignature(signature, eventName, parameterTypes, parameterNames, contractAddr string) error {
	now := time.Now()

	// Check if signature already exists
	var existingID int64
	var existingParamTypes, existingContractList string
	err := d.db.QueryRow("SELECT id, parameter_types, contract_list FROM event_signatures WHERE signature = ?", signature).Scan(&existingID, &existingParamTypes, &existingContractList)

	if err == sql.ErrNoRows {
		// New signature - insert
		_, err = d.db.Exec(`
			INSERT INTO event_signatures (signature, event_name, parameter_types, parameter_names, first_seen, last_seen, usage_count, contract_list)
			VALUES (?, ?, ?, ?, ?, ?, 1, ?)
		`, signature, eventName, parameterTypes, parameterNames, now, now, fmt.Sprintf(`["%s"]`, contractAddr))
		if err != nil {
			return fmt.Errorf("failed to insert event signature: %v", err)
		}

		// Get the inserted ID
		var id int64
		err = d.db.QueryRow("SELECT id FROM event_signatures WHERE signature = ?", signature).Scan(&id)
		if err != nil {
			return fmt.Errorf("failed to get inserted signature ID: %v", err)
		}

		// Insert contract usage
		_, err = d.db.Exec(`
			INSERT INTO contract_usage (signature_id, contract_addr, first_seen, last_seen, usage_count)
			VALUES (?, ?, ?, ?, 1)
		`, id, contractAddr, now, now)
		if err != nil {
			return fmt.Errorf("failed to insert contract usage: %v", err)
		}

		return nil
	} else if err != nil {
		return fmt.Errorf("failed to query existing signature: %v", err)
	}

	// Signature exists - check if parameter types are different
	if existingParamTypes != parameterTypes {
		// Different parameter types - create new entry with modified signature
		newSignature := fmt.Sprintf("%s_%d", signature, now.Unix())
		_, err = d.db.Exec(`
			INSERT INTO event_signatures (signature, event_name, parameter_types, parameter_names, first_seen, last_seen, usage_count, contract_list)
			VALUES (?, ?, ?, ?, ?, ?, 1, ?)
		`, newSignature, eventName, parameterTypes, parameterNames, now, now, fmt.Sprintf(`["%s"]`, contractAddr))
		if err != nil {
			return fmt.Errorf("failed to insert new signature variant: %v", err)
		}

		// Get the new ID
		var newID int64
		err = d.db.QueryRow("SELECT id FROM event_signatures WHERE signature = ?", newSignature).Scan(&newID)
		if err != nil {
			return fmt.Errorf("failed to get new signature ID: %v", err)
		}

		// Insert contract usage for new signature
		_, err = d.db.Exec(`
			INSERT INTO contract_usage (signature_id, contract_addr, first_seen, last_seen, usage_count)
			VALUES (?, ?, ?, ?, 1)
		`, newID, contractAddr, now, now)
		if err != nil {
			return fmt.Errorf("failed to insert contract usage for new signature: %v", err)
		}

		return nil
	}

	// Same signature and parameter types - update existing
	_, err = d.db.Exec(`
		UPDATE event_signatures 
		SET last_seen = ?, usage_count = usage_count + 1
		WHERE id = ?
	`, now, existingID)
	if err != nil {
		return fmt.Errorf("failed to update existing signature: %v", err)
	}

	// Update or insert contract usage
	_, err = d.db.Exec(`
		INSERT INTO contract_usage (signature_id, contract_addr, first_seen, last_seen, usage_count)
		VALUES (?, ?, ?, ?, 1)
		ON CONFLICT(signature_id, contract_addr) DO UPDATE SET
			last_seen = excluded.last_seen,
			usage_count = contract_usage.usage_count + 1
	`, existingID, contractAddr, now, now)
	if err != nil {
		return fmt.Errorf("failed to update contract usage: %v", err)
	}

	return nil
}

// GetEventSignatures retrieves all event signatures
func (d *Database) GetEventSignatures() ([]EventSignature, error) {
	rows, err := d.db.Query(`
		SELECT id, signature, event_name, parameter_types, parameter_names, first_seen, last_seen, usage_count, contract_list
		FROM event_signatures
		ORDER BY last_seen DESC
	`)
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

// GetEventSignatureByHash retrieves an event signature by its hash
func (d *Database) GetEventSignatureByHash(signature string) (*EventSignature, error) {
	var sig EventSignature
	var firstSeenStr, lastSeenStr string
	err := d.db.QueryRow(`
		SELECT id, signature, event_name, parameter_types, parameter_names, first_seen, last_seen, usage_count, contract_list
		FROM event_signatures
		WHERE signature = ?
	`, signature).Scan(&sig.ID, &sig.Signature, &sig.EventName, &sig.ParameterTypes, &sig.ParameterNames, &firstSeenStr, &lastSeenStr, &sig.UsageCount, &sig.ContractList)

	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to query event signature: %v", err)
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

	return &sig, nil
}

// CleanupUnknownEvents removes all unknown events from the database
func (d *Database) CleanupUnknownEvents() error {
	// Delete unknown events
	result, err := d.db.Exec("DELETE FROM event_signatures WHERE event_name LIKE 'unknown_event%'")
	if err != nil {
		return fmt.Errorf("failed to delete unknown events: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	log.Printf("Cleaned up %d unknown events from database", rowsAffected)
	return nil
}

// MigrateOldDatabase migrates the old database structure to the new one
func (d *Database) MigrateOldDatabase() error {
	// Check if we need to migrate (if parameter_names column doesn't exist)
	var columnExists bool
	err := d.db.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM pragma_table_info('event_signatures') 
		WHERE name = 'parameter_names'
	`).Scan(&columnExists)

	if err != nil {
		return fmt.Errorf("failed to check column existence: %v", err)
	}

	if columnExists {
		log.Println("Database already migrated, skipping migration")
		return nil
	}

	log.Println("Migrating database structure...")

	// Add parameter_names column
	_, err = d.db.Exec("ALTER TABLE event_signatures ADD COLUMN parameter_names TEXT DEFAULT '[]'")
	if err != nil {
		return fmt.Errorf("failed to add parameter_names column: %v", err)
	}

	// Rename parameters column to parameter_types
	_, err = d.db.Exec("ALTER TABLE event_signatures RENAME COLUMN parameters TO parameter_types")
	if err != nil {
		return fmt.Errorf("failed to rename parameters column: %v", err)
	}

	log.Println("Database migration completed successfully")
	return nil
}
