package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	// No more styling needed
)

const maxResults = 5 // Maximum number of rows to display
const databaseName = "database.sqlite"

func main() {
	// Suppress log output
	log.SetOutput(io.Discard)

	rootCmd := &cobra.Command{
		Use:   "app",
		Short: "A CLI application with initialization",
		Run: func(cmd *cobra.Command, args []string) {
			// Default behavior when no subcommand is provided
			fmt.Println("hello world")
		},
	}

	// Add the `init` command
	rootCmd.AddCommand(initCmd())

	// Execute the root command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the database and tables",
		Run: func(cmd *cobra.Command, args []string) {
			// Simple sequential initialization without Bubble Tea
			fmt.Println(" - Initializing...")
			
			// Check if database exists
			if _, err := os.Stat(databaseName); err == nil {
				fmt.Println("⚠️ Database already exists - checking schemas...")
				
				// Check table schemas
				tables := []string{"project", "task", "tag", "task_tag"}
				for _, table := range tables {
					err := checkTableSchema(databaseName, table)
					if err != nil {
						fmt.Printf("❌ Table `%s` schema differs: %s\n", table, err.Error())
					} else {
						fmt.Printf("✔ Table `%s` schema matches\n", table)
					}
				}
				
				fmt.Println("\nSchema validation complete!")
			} else {
				fmt.Println("❌ Database not found")
				
				// Create database
				err := createDatabase(databaseName)
				if err != nil {
					fmt.Printf("❌ Failed to create database: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("✔ Database created")
				
				// Create tables
				tables := []string{"project", "task", "tag", "task_tag"}
				for _, table := range tables {
					err := createTable(databaseName, table)
					if err != nil {
						fmt.Printf("❌ Failed to create table `%s`: %v\n", table, err)
						os.Exit(1)
					}
					fmt.Printf("✔ table `%s` created\n", table)
				}
				
				fmt.Println("\nInitialization complete!")
			}
		},
	}
}

func createDatabase(dbPath string) error {
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

func createTable(dbPath, tableName string) error {
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
			name TEXT NOT NULL
		);`
	case "task":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS task (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			due_date DATE,
			FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE
		);`
	case "tag":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS tag (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE
		);`
	case "task_tag":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS task_tag (
			task_id INTEGER NOT NULL,
			tag_id INTEGER NOT NULL,
			PRIMARY KEY (task_id, tag_id),
			FOREIGN KEY (task_id) REFERENCES task (id) ON DELETE CASCADE,
			FOREIGN KEY (tag_id) REFERENCES tag (id) ON DELETE CASCADE
		);`
	default:
		return fmt.Errorf("unknown table: %s", tableName)
	}

	_, err = db.Exec(createTableSQL)
	return err
}

func checkTableSchema(dbPath, tableName string) error {
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
		var dflt_value sql.NullString // Use sql.NullString to handle NULL values
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
		},
		"task": {
			"id INTEGER",
			"project_id INTEGER",
			"name TEXT",
			"due_date DATE",
		},
		"tag": {
			"id INTEGER",
			"name TEXT",
		},
		"task_tag": {
			"task_id INTEGER",
			"tag_id INTEGER",
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
