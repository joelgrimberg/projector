package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/joelgrimberg/projector/database"
	"github.com/joelgrimberg/projector/models"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxResults = 5 // Maximum number of rows to display

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render
	mainStyle = lipgloss.NewStyle().MarginLeft(1)
)

// Model represents the UI state
type Model struct {
	spinner    spinner.Model
	results    []models.Result
	quitting   bool
	step       int
	tableIndex int  // Track which table we're creating/checking
	schemaMode bool // True if we're checking schemas, false if creating tables
}

// NewModel creates a new UI model
func NewModel() Model {
	sp := spinner.New()
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("206"))

	// Prefill the results slice with dots
	prefilledResults := make([]models.Result, maxResults)
	for i := 0; i < maxResults; i++ {
		prefilledResults[i] = models.Result{Emoji: "•", Message: "........................"}
	}

	return Model{
		spinner:    sp,
		results:    prefilledResults,
		step:       0,
		tableIndex: 0,
		schemaMode: false,
	}
}

// Init initializes the UI model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		runInitStep(),
	)
}

// Update handles UI updates
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.quitting = true
		return m, tea.Quit

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case models.Result:
		// Add the new result and shift existing messages up
		m.results = append(m.results[1:], msg)
		m.step++

		// Check if we're entering schema mode (database already existed)
		if m.step == 1 && strings.Contains(msg.Message, "Database already exists") {
			m.schemaMode = true
		}

		// Check if schema validation failed
		if m.schemaMode && strings.Contains(msg.Message, "schema differs") {
			// Abort initialization due to schema mismatch
			return m, tea.Quit
		}

		// Continue with next step based on current step
		switch m.step {
		case 1: // After database check/creation, start processing tables
			m.tableIndex = 0
			// Check if we're in schema mode (database already existed)
			if m.schemaMode {
				return m, checkTableSchemaStep(m.tableIndex)
			} else {
				return m, createTableStep(m.tableIndex)
			}
		case 2, 3, 4, 5, 6, 7: // Continue processing tables (6 steps total due to status seeding/verification)
			if m.step == 3 && m.tableIndex == 1 { // Special case: status table seeding or verification
				if m.schemaMode {
					return m, verifyStatusTableStep()
				} else {
					return m, seedStatusTableStep()
				}
			} else if m.tableIndex < 4 { // 5 tables total (0-4)
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

// View renders the UI
func (m Model) View() string {
	s := "\n" + m.spinner.View() + " Initializing...\n\n"

	// Render the results slice
	for _, res := range m.results {
		s += fmt.Sprintf("%s %s\n", res.Emoji, res.Message)
	}

	// Check if initialization was aborted due to schema differences
	abortedDueToSchema := false
	for _, res := range m.results {
		if strings.Contains(res.Message, "schema differs") {
			abortedDueToSchema = true
			break
		}
	}

	if abortedDueToSchema {
		// Show abort message when schema validation failed
		s += "\n❌ Initialization aborted due to schema differences!\n"
	} else if m.step >= 7 && m.tableIndex >= 4 {
		// Show success message when all tables are processed (6 steps total due to status seeding)
		s += "\n🎉 Initialization complete!\n"
	} else {
		// Only show "Press any key to exit" when initialization is still in progress
		s += helpStyle("\nPress any key to exit\n")
	}

	if m.quitting {
		s += "\n"
	}

	return mainStyle.Render(s)
}

// runInitStep handles the initial database check/creation
func runInitStep() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)

		// Check if database exists
		if database.DatabaseExists(database.GetDatabasePath()) {
			// Database exists, check schemas instead of creating
			return models.Result{Emoji: "⚠️", Message: "Database already exists, checking schemas..."}
		} else {
			// Database doesn't exist, create it
			err := database.CreateDatabase(database.GetDatabasePath())
			if err != nil {
				return models.Result{Emoji: "❌", Message: "Failed to create database"}
			}
			return models.Result{Emoji: "🗃️", Message: "Database created"}
		}
	}
}

// createTableStep creates one table at a time
func createTableStep(tableIndex int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)

		tables := []string{"project", "status", "action", "tag", "action_tag"}
		table := tables[tableIndex]

		err := database.CreateTable(database.GetDatabasePath(), table)
		if err != nil {
			return models.Result{Emoji: "❌", Message: fmt.Sprintf("Failed to create table `%s`", table)}
		}

		// If this is the status table, show creation message first
		if table == "status" {
			return models.Result{Emoji: "📝", Message: "Table `status` created"}
		}

		// Use 🚀 for project and action tables, 🏷️ for tag table, 🧩 for action_tag table
		if table == "project" || table == "action" {
			return models.Result{Emoji: "🚀", Message: fmt.Sprintf("Table `%s` created", table)}
		}
		if table == "tag" {
			return models.Result{Emoji: "🏷️", Message: fmt.Sprintf("Table `%s` created", table)}
		}
		if table == "action_tag" {
			return models.Result{Emoji: "🧩", Message: fmt.Sprintf("Table `%s` created", table)}
		}

		return models.Result{Emoji: "✔", Message: fmt.Sprintf("Table `%s` created", table)}
	}
}

// checkTableSchemaStep checks one table schema at a time
func checkTableSchemaStep(tableIndex int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(1 * time.Second)

		tables := []string{"project", "status", "action", "tag", "action_tag"}
		table := tables[tableIndex]

		err := database.CheckTableSchema(database.GetDatabasePath(), table)
		if err != nil {
			// Get both schemas for comparison
			expectedSchema := database.GetExpectedSchema(table)
			actualSchema := database.GetActualSchema(database.GetDatabasePath(), table)

			return models.Result{
				Emoji: "❌",
				Message: fmt.Sprintf("Table `%s` schema differs:\nExpected: %s\nActual: %s",
					table, expectedSchema, actualSchema),
			}
		}

		return models.Result{Emoji: "✔", Message: fmt.Sprintf("Table `%s` schema matches", table)}
	}
}

// seedStatusTableStep shows the status table seeding message
func seedStatusTableStep() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)
		return models.Result{Emoji: "🌱", Message: "Table `status` seeded"}
	}
}

// verifyStatusTableStep verifies the status table data
func verifyStatusTableStep() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(500 * time.Millisecond)

		isValid, err := database.VerifyStatusTableData(database.GetDatabasePath())
		if err != nil {
			return models.Result{Emoji: "❌", Message: fmt.Sprintf("Failed to verify status table data: %v", err)}
		}

		if isValid {
			return models.Result{Emoji: "🌱", Message: "Table `status` data verified"}
		} else {
			return models.Result{Emoji: "⚠️", Message: "Table `status` data incomplete"}
		}
	}
}
