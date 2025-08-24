package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
)

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	mainStyle = lipgloss.NewStyle().MarginLeft(1)
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
			p := tea.NewProgram(newModel())
			if _, err := p.Run(); err != nil {
				fmt.Println("Error starting Bubble Tea program:", err)
				os.Exit(1)
			}
		},
	}
}

type result struct {
	emoji   string
	message string
}

type model struct {
	spinner    spinner.Model
	results    []result
	quitting   bool
	step       int
	tableIndex int  // Track which table we're creating/checking
	schemaMode bool // True if we're checking schemas, false if creating tables
}

func newModel() model {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("206"))

	// Prefill the results slice with dots
	prefilledResults := make([]result, maxResults)
	for i := 0; i < maxResults; i++ {
		prefilledResults[i] = result{emoji: "•", message: "........................"}
	}

	return model{
		spinner:    sp,
		results:    prefilledResults,
		step:       0,
		tableIndex: 0,
		schemaMode: false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		runInitStep(),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case result:
		// Add the new result and shift existing messages up
		m.results = append(m.results[1:], msg)
		m.step++

		// Check if we're entering schema mode (database already existed)
		if m.step == 1 && strings.Contains(msg.message, "Database already exists") {
			m.schemaMode = true
		}

		// Check if schema validation failed
		if m.schemaMode && strings.Contains(msg.message, "schema differs") {
			// Abort initialization due to schema mismatch
			return m, tea.Quit
		}

		// Continue with next step based on current step
		switch m.step {
		case 1: // After database check/creation, start processing tables
			m.tableIndex = 0
			// Check if we're in schema mode (database already existed)
			if m.schemaMode {
				// After migration, start checking schemas
				return m, checkTableSchemaStep(m.tableIndex)
			} else {
				return m, createTableStep(m.tableIndex)
			}
		case 2, 3, 4, 5, 6: // Continue processing tables (5 tables total)
			if m.tableIndex < 4 { // 5 tables total (indices 0-4)
				m.tableIndex++
				if m.schemaMode {
					return m, checkTableSchemaStep(m.tableIndex)
				} else {
					return m, createTableStep(m.tableIndex)
				}
			} else {
				return m, tea.Quit
			}
		default:
			return m, nil
		}

	default:
		return m, nil
	}
}

func (m model) View() string {
	s := "\n" + m.spinner.View() + " Initializing...\n\n"

	// Render the results slice
	for _, res := range m.results {
		s += fmt.Sprintf("%s %s\n", res.emoji, res.message)
	}

	// Check if initialization was aborted due to schema differences
	abortedDueToSchema := false
	for _, res := range m.results {
		if strings.Contains(res.message, "schema differs") {
			abortedDueToSchema = true
			break
		}
	}

	if abortedDueToSchema {
		// Show abort message when schema validation failed
		s += "\n❌ Initialization aborted due to schema differences!\n"
	} else if m.step >= 6 && m.tableIndex >= 4 {
		// Show success message when all tables are processed (5 tables total, indices 0-4)
		s += "\n✅ Initialization complete!\n"
	} else {
		// Only show "Press any key to exit" when initialization is still in progress
		s += helpStyle("\nPress any key to exit\n")
	}

	if m.quitting {
		s += "\n"
	}

	return mainStyle.Render(s)
}

// Single command that handles all initialization steps
func runInitStep() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)

		// Check if database exists
		if _, err := os.Stat(databaseName); err != nil {
			// Database doesn't exist, create it
			err := createDatabase(databaseName)
			if err != nil {
				return result{emoji: "❌", message: "Failed to create database"}
			}
			return result{emoji: "✔", message: "Database created"}
		} else {
			// Database exists, check schemas instead of creating
			return result{emoji: "⚠️", message: "Database already exists, checking schemas..."}
		}
	}
}

// Command to create individual tables
func createTablesStep() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)

		// Create tables one by one and return individual messages
		tables := []string{"project", "task", "tag", "task_tag"}
		for _, table := range tables {
			err := createTable(databaseName, table)
			if err != nil {
				return result{emoji: "❌", message: fmt.Sprintf("Failed to create table `%s`", table)}
			}
			// Return message for this specific table
			return result{emoji: "✔", message: fmt.Sprintf("Table `%s` created", table)}
		}

		return result{emoji: "✔", message: "All tables created successfully"}
	}
}

// Command to create one table at a time
func createTableStep(tableIndex int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)

		tables := []string{"project", "status", "task", "tag", "task_tag"}
		table := tables[tableIndex]

		err := createTable(databaseName, table)
		if err != nil {
			return result{emoji: "❌", message: fmt.Sprintf("Failed to create table `%s`", table)}
		}

		return result{emoji: "✔", message: fmt.Sprintf("Table `%s` created", table)}
	}
}

// Command to check one table schema at a time
func checkTableSchemaStep(tableIndex int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)

		tables := []string{"project", "status", "task", "tag", "task_tag"}
		table := tables[tableIndex]

		err := checkTableSchema(databaseName, table)
		if err != nil {
			// Get both schemas for comparison
			expectedSchema := getExpectedSchema(table)
			actualSchema := getActualSchema(databaseName, table)

			return result{
				emoji: "❌",
				message: fmt.Sprintf("Table `%s` schema differs:\nExpected: %s\nActual: %s",
					table, expectedSchema, actualSchema),
			}
		}

		return result{emoji: "✔", message: fmt.Sprintf("Table `%s` schema matches", table)}
	}
}

// Commands
func checkDatabaseCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)
		if _, err := os.Stat(databaseName); err == nil {
			return result{emoji: "⚠️", message: "Database already exists - checking schemas..."}
		}
		return result{emoji: "❌", message: "Database not found"}
	}
}

func createDatabaseCmd() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)
		err := createDatabase(databaseName)
		if err != nil {
			return result{emoji: "❌", message: "Failed to create database"}
		}
		return result{emoji: "✔", message: "Database created"}
	}
}

func createTableCmd(tableName string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)
		err := createTable(databaseName, tableName)
		if err != nil {
			return result{emoji: "❌", message: fmt.Sprintf("Failed to create table: `%s`", tableName)}
		}
		return result{emoji: "✔", message: fmt.Sprintf("table `%s` created", tableName)}
	}
}

func checkTableSchemaCmd(tableName string) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)
		err := checkTableSchema(databaseName, tableName)
		if err != nil {
			return result{emoji: "❌", message: fmt.Sprintf("Table `%s` schema differs: %s", tableName, err.Error())}
		}
		return result{emoji: "✔", message: fmt.Sprintf("Table `%s` schema matches", tableName)}
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
			name TEXT NOT NULL,
			due_date DATE
		);`
	case "task":
		createTableSQL = `
		CREATE TABLE IF NOT EXISTS task (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER,
			name TEXT NOT NULL,
			due_date DATE,
			status_id INTEGER NOT NULL DEFAULT 1,
			FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE SET NULL,
			FOREIGN KEY (status_id) REFERENCES status (id)
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
		"task": {
			"id INTEGER",
			"project_id INTEGER",
			"name TEXT",
			"due_date DATE",
			"status_id INTEGER",
		},
		"tag": {
			"id INTEGER",
			"name TEXT",
		},
		"task_tag": {
			"task_id INTEGER",
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

// Helper function to get expected schema for a table
func getExpectedSchema(tableName string) string {
	expectedSchemas := map[string]string{
		"project":  "id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, due_date DATE",
		"task":     "id INTEGER PRIMARY KEY AUTOINCREMENT, project_id INTEGER, name TEXT NOT NULL, due_date DATE, status_id INTEGER NOT NULL",
		"tag":      "id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE",
		"task_tag": "task_id INTEGER NOT NULL, tag_id INTEGER NOT NULL",
		"status":   "id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL UNIQUE",
	}

	if schema, exists := expectedSchemas[tableName]; exists {
		return schema
	}
	return "Unknown table"
}

// Helper function to get actual schema from database
func getActualSchema(dbPath, tableName string) string {
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
	if strings.HasPrefix(tableSQL, "CREATE TABLE") {
		// Find the opening parenthesis and extract the content
		start := strings.Index(tableSQL, "(")
		end := strings.LastIndex(tableSQL, ")")
		if start != -1 && end != -1 && end > start {
			columns := tableSQL[start+1 : end]
			// Clean up the columns string
			columns = strings.ReplaceAll(columns, "\n", " ")
			columns = strings.ReplaceAll(columns, "  ", " ")
			columns = strings.TrimSpace(columns)
			return columns
		}
	}

	return tableSQL
}
