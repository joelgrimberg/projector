package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	_ "github.com/mattn/go-sqlite3"
)

type errMsg error

type model struct {
	spinner  spinner.Model
	quitting bool
	err      error
	done     bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{spinner: s}
}

func (m model) Init() tea.Cmd {
	// Start the spinner and the database initialization
	return tea.Batch(m.spinner.Tick, initializeDatabaseAsync("example.db"))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	case errMsg:
		m.err = msg
		return m, tea.Quit

	case bool:
		m.done = msg
		return m, tea.Quit

	default:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n\n   Error: %v\n\n", m.err)
	}
	if m.done {
		return "\n\n   Database initialized successfully!\n\n"
	}
	str := fmt.Sprintf("\n\n   %s Initializing database...press q to quit\n\n", m.spinner.View())
	if m.quitting {
		return str + "\n"
	}
	return str
}

func initializeDatabaseAsync(dbPath string) tea.Cmd {
	return func() tea.Msg {
		// Simulate database initialization
		time.Sleep(2 * time.Second) // Simulate delay for spinner demonstration

		db, err := sql.Open("sqlite3", dbPath)
		if err != nil {
			return errMsg(err)
		}
		defer db.Close()

		createTableSQL := `
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			age INTEGER
		);`
		_, err = db.Exec(createTableSQL)
		if err != nil {
			return errMsg(err)
		}

		return true // Signal that the operation is done
	}
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
