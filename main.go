package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
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

func main() {
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

type initState int

const (
	stateCheckForDatabase initState = iota
	stateCreateDatabase
	stateCheckForTables
	stateCreateTables
	stateDone
)

type result struct {
	state   initState
	success bool
	message string
}

type model struct {
	spinner  spinner.Model
	results  []string
	state    initState
	quitting bool
}

func newModel() model {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("206"))

	return model{
		spinner: sp,
		results: []string{"Initializing...\n"},
		state:   stateCheckForDatabase,
	}
}

func (m model) Init() tea.Cmd {
	log.Println("Starting initialization...")
	return tea.Batch(
		m.spinner.Tick,
		runInitialization(stateCheckForDatabase),
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
		m.results = append(m.results, msg.message)
		switch msg.state {
		case stateCheckForDatabase:
			if msg.success {
				m.state = stateCheckForTables
				return m, runInitialization(stateCheckForTables)
			}
			m.state = stateCreateDatabase
			return m, runInitialization(stateCreateDatabase)

		case stateCreateDatabase:
			if msg.success {
				m.state = stateCreateTables
				return m, runInitialization(stateCreateTables)
			}

		case stateCheckForTables:
			if msg.success {
				m.state = stateDone
				return m, tea.Quit
			}
			m.state = stateCreateTables
			return m, runInitialization(stateCreateTables)

		case stateCreateTables:
			if msg.success {
				m.state = stateDone
				return m, tea.Quit
			}
		}
		return m, nil

	default:
		return m, nil
	}
}

func (m model) View() string {
	s := "\n" + m.spinner.View() + " " + m.results[0] // Show "Initializing..." only once

	for _, res := range m.results[1:] {
		s += res + "\n"
	}

	if m.state == stateDone {
		s += "\nInitialization complete!\n"
	}

	s += helpStyle("\nPress any key to exit\n")

	if m.quitting {
		s += "\n"
	}

	return mainStyle.Render(s)
}

func runInitialization(state initState) tea.Cmd {
	return func() tea.Msg {
		switch state {
		case stateCheckForDatabase:
			if _, err := os.Stat("example.db"); err == nil {
				return result{state: state, success: true, message: "- Database already exists"}
			}
			// Database not found
			return result{state: state, success: false, message: "- Database not found"}

		case stateCreateDatabase:
			// Message: Creating database
			time.Sleep(1 * time.Second) // Simulate delay
			err := createDatabase("example.db")
			if err != nil {
				return result{state: state, success: false, message: "- Failed to create database"}
			}
			return result{state: state, success: true, message: "✔ Database created. Creating tables..."}

		case stateCheckForTables:
			// Message: Checking if tables exist
			tablesExist, err := checkTablesExist("example.db")
			if err != nil {
				return result{state: state, success: false, message: "- Failed to check tables"}
			}
			if tablesExist {
				return result{state: state, success: true, message: "✔ All tables already exist"}
			}
			return result{state: state, success: false, message: "- Tables do not exist"}

		case stateCreateTables:
			// Message: Creating tables
			messages, err := createTables("example.db")
			if err != nil {
				return result{state: state, success: false, message: "- Failed to create tables"}
			}
			return result{state: state, success: true, message: messages}
		}

		return nil
	}
}

func createDatabase(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}

func createTables(dbPath string) (string, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()

	messages := ""

	// Create the `project` table
	messages += "- Creating table: project\n"
	createProjectTableSQL := `
	CREATE TABLE IF NOT EXISTS project (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL
	);`
	_, err = db.Exec(createProjectTableSQL)
	if err != nil {
		return "", err
	}
	messages += "✔ Created table: project\n"

	// Create the `task` table
	messages += "- Creating table: task\n"
	createTaskTableSQL := `
	CREATE TABLE IF NOT EXISTS task (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		due_date DATE,
		FOREIGN KEY (project_id) REFERENCES project (id) ON DELETE CASCADE
	);`
	_, err = db.Exec(createTaskTableSQL)
	if err != nil {
		return "", err
	}
	messages += "✔ Created table: task\n"

	// Create the `tag` table
	messages += "- Creating table: tag\n"
	createTagTableSQL := `
	CREATE TABLE IF NOT EXISTS tag (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE
	);`
	_, err = db.Exec(createTagTableSQL)
	if err != nil {
		return "", err
	}
	messages += "✔ Created table: tag\n"

	// Create the `task_tag` table
	messages += "- Creating table: task_tag\n"
	createTaskTagTableSQL := `
	CREATE TABLE IF NOT EXISTS task_tag (
		task_id INTEGER NOT NULL,
		tag_id INTEGER NOT NULL,
		PRIMARY KEY (task_id, tag_id),
		FOREIGN KEY (task_id) REFERENCES task (id) ON DELETE CASCADE,
		FOREIGN KEY (tag_id) REFERENCES tag (id) ON DELETE CASCADE
	);`
	_, err = db.Exec(createTaskTagTableSQL)
	if err != nil {
		return "", err
	}
	messages += "✔ Created table: task_tag\n"

	return messages, nil
}

func checkTablesExist(dbPath string) (bool, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()

	requiredTables := []string{"project", "task", "tag", "task_tag"}
	for _, table := range requiredTables {
		var name string
		err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?;", table).Scan(&name)
		if err == sql.ErrNoRows {
			return false, nil // Table does not exist
		} else if err != nil {
			return false, err // Some other error occurred
		}
	}
	return true, nil // All tables exist
}
