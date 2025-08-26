package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	_ "github.com/mattn/go-sqlite3"
)

const DatabaseName = "projector.db"

// GetDatabasePath returns the proper database path in ~/.local/share/projector/
func GetDatabasePath() string {
	// Check for environment variable override
	if envPath := os.Getenv("PROJECTOR_DB_PATH"); envPath != "" {
		return envPath
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		return DatabaseName
	}

	// Use ~/.local/share/projector/ for all platforms
	dbDir := filepath.Join(homeDir, ".local", "share", "projector")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		// Fallback to current directory
		return DatabaseName
	}

	return filepath.Join(dbDir, DatabaseName)
}

// CreateDatabase creates a new SQLite database file
func CreateDatabase(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Test if the database connection works
	if err := db.Ping(); err != nil {
		return err
	}

	// Create a simple table to ensure the database file is written to disk
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS _init_check (id INTEGER PRIMARY KEY);")
	if err != nil {
		return err
	}

	// Drop the temporary table
	_, err = db.Exec("DROP TABLE _init_check;")
	if err != nil {
		return err
	}

	return nil
}

// CreateTable creates a specific table in the database
func CreateTable(dbPath, tableName string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	var createTableSQL string
	switch tableName {
	case "project":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS project (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			due_date DATE
		);`
	case "action":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS action (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER,
			name TEXT NOT NULL,
			note TEXT,
			due_date DATE,
			status_id INTEGER NOT NULL DEFAULT 1,
			repeat_count INTEGER DEFAULT 0,
			repeat_interval TEXT,
			repeat_pattern TEXT,
			repeat_until DATE,
			parent_action_id INTEGER,
			FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE SET NULL,
			FOREIGN KEY (status_id) REFERENCES status (id),
			FOREIGN KEY (parent_action_id) REFERENCES action (id) ON DELETE SET NULL
		);`
	case "tag":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS tag (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);`
	case "action_tag":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS action_tag (
			action_id INTEGER NOT NULL,
			tag_id INTEGER NOT NULL,
			PRIMARY KEY (action_id, tag_id),
			FOREIGN KEY (action_id) REFERENCES action (id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id) REFERENCES tag (id) ON DELETE CASCADE
		);`
	case "status":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS status (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);`
	default:
		return fmt.Errorf("unknown table: %s", tableName)
	}

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return err
	}

	// If this is the status table, insert the default statuses
	if tableName == "status" {
		insertStatusSQL := `
		INSERT OR IGNORE INTO status (id, name) VALUES 
		(1, 'todo'),
		(2, 'done');`
		_, err = db.Exec(insertStatusSQL)
		if err != nil {
			return err
		}
	}

	return nil
}

// CheckTableSchema validates that a table has the expected schema
func CheckTableSchema(dbPath, tableName string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if table exists
	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s';", tableName)).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		return fmt.Errorf("table `%s` not found", tableName)
	}

	// Get the actual table schema
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info('%s');", tableName))
	if err != nil {
		return err
	}
	defer rows.Close()

	var actualColumns []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull int
		var dflt_value sql.NullString
		var pk int
		err := rows.Scan(&cid, &name, &typ, &notnull, &dflt_value, &pk)
		if err != nil {
			return err
		}
		actualColumns = append(actualColumns, fmt.Sprintf("%s %s", name, typ))
	}

	// Define expected schemas
	expectedSchemas := map[string][]string{
		"project": {
			"id INTEGER",
			"name TEXT",
			"due_date DATE",
		},
		"action": {
			"id INTEGER",
			"project_id INTEGER",
			"name TEXT",
			"note TEXT",
			"due_date DATE",
			"status_id INTEGER",
			"repeat_count INTEGER",
			"repeat_interval TEXT",
			"repeat_pattern TEXT",
			"repeat_until DATE",
			"parent_action_id INTEGER",
		},
		"tag": {
			"id INTEGER",
			"name TEXT",
		},
		"action_tag": {
			"action_id INTEGER",
			"tag_id INTEGER",
		},
		"status": {
			"id INTEGER",
			"name TEXT",
		},
	}

	expectedColumns := expectedSchemas[tableName]
	if len(expectedColumns) == 0 {
		return fmt.Errorf("unknown table: %s", tableName)
	}

	// Compare schemas
	if len(actualColumns) != len(expectedColumns) {
		return fmt.Errorf("table `%s` schema differs: expected %d columns, got %d", tableName, len(expectedColumns), len(actualColumns))
	}

	// For now, just check column count and basic structure
	// In a real implementation, you might want to do more detailed schema comparison
	return nil
}

// GetExpectedSchema returns the expected schema string for a table
func GetExpectedSchema(tableName string) string {
	expectedSchemas := map[string]string{
		"project":  "id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, due_date DATE",
		"action":     "id INTEGER PRIMARY KEY AUTOINCREMENT, project_id INTEGER, name TEXT NOT NULL, note TEXT, due_date DATE, status_id INTEGER NOT NULL, repeat_count INTEGER DEFAULT 0, repeat_interval TEXT, repeat_pattern TEXT, repeat_until DATE, parent_action_id INTEGER",
		"tag":      "id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE",
		"action_tag": "action_id INTEGER NOT NULL, tag_id INTEGER NOT NULL, PRIMARY KEY (action_id, tag_id), FOREIGN KEY (action_id) REFERENCES action (id) ON DELETE CASCADE, FOREIGN KEY (tag_id) REFERENCES tag (id) ON DELETE CASCADE",
		"status":   "id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE",
	}

	if schema, exists := expectedSchemas[tableName]; exists {
		return schema
	}
	return "Unknown table"
}

// GetActualSchema returns the actual schema from database
func GetActualSchema(dbPath, tableName string) string {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Sprintf("Error opening database: %v", err)
	}
	defer db.Close()

	// Check if table exists
	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='%s';", tableName)).Scan(&count)
	if err != nil {
		return fmt.Sprintf("Error checking table existence: %v", err)
	}

	if count == 0 {
		return "Table not found"
	}

	// Get the complete table definition from sqlite_master
	var tableSQL string
	err = db.QueryRow(fmt.Sprintf("SELECT sql FROM sqlite_master WHERE type='table' AND name='%s';", tableName)).Scan(&tableSQL)
	if err != nil {
		return fmt.Sprintf("Error getting table definition: %v", err)
	}

	// Extract just the column definitions from the CREATE TABLE statement
	// Remove the CREATE TABLE part and keep just the column definitions
	if len(tableSQL) > 0 && tableSQL[:12] == "CREATE TABLE" {
		// Find the opening parenthesis and extract the content
		start := 0
		for i, char := range tableSQL {
			if char == '(' {
				start = i
				break
			}
		}
		end := 0
		for i := len(tableSQL) - 1; i >= 0; i-- {
			if tableSQL[i] == ')' {
				end = i
				break
			}
		}
		if start != -1 && end != -1 && end > start {
			columns := tableSQL[start+1 : end]
			// Clean up the columns string
			columns = fmt.Sprintf("%s", columns)
			return columns
		}
	}

	return tableSQL
}

// DatabaseExists checks if the database file exists
func DatabaseExists(dbPath string) bool {
	_, err := os.Stat(dbPath)
	return err == nil
}

// DeleteProject deletes a project from the database
func DeleteProject(dbPath string, projectID uint) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Check if project exists
	project, err := GetProjectByID(dbPath, projectID)
	if err != nil {
		return fmt.Errorf("error checking project existence: %v", err)
	}
	if project == nil {
		return fmt.Errorf("project not found")
	}

	// Delete the project
	query := "DELETE FROM project WHERE id = ?"
	_, err = db.Exec(query, projectID)
	if err != nil {
		return fmt.Errorf("failed to delete project: %v", err)
	}

	return nil
}

// VerifyStatusTableData checks if the status table contains the expected initial data
func VerifyStatusTableData(dbPath string) (bool, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return false, fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Check if the expected statuses exist
	query := `
		SELECT COUNT(*) FROM status 
		WHERE (id = 1 AND name = 'todo') 
		   OR (id = 2 AND name = 'done')`

	var count int
	err = db.QueryRow(query).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to verify status data: %v", err)
	}

	// Should have exactly 2 statuses (todo and done)
	return count == 2, nil
}
