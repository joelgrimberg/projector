package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"main/api"
	"main/database"
	"main/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

func main() {
	// Suppress log output
	log.SetOutput(io.Discard)

	rootCmd := &cobra.Command{
		Use:   "projector",
		Short: "A CLI application for project and task management",
		Run: func(cmd *cobra.Command, args []string) {
			// Default behavior when no subcommand is provided
			startAPIServer()
		},
	}

	// Add the `init` command
	rootCmd.AddCommand(initCmd())

	// Add the `migrate` command
	rootCmd.AddCommand(migrateCmd())

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
			p := tea.NewProgram(ui.NewModel())
			if _, err := p.Run(); err != nil {
				fmt.Println("Error starting Bubble Tea program:", err)
				os.Exit(1)
			}
		},
	}
}

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Migrate database schema to add note field to tasks",
		Run: func(cmd *cobra.Command, args []string) {
			runMigration()
		},
	}
}

func runMigration() {
	fmt.Println("🔄 Starting database migration...")

	// Check if database exists
	if !database.DatabaseExists(database.DatabaseName) {
		fmt.Println("❌ Database not found. Please run 'projector init' first.")
		return
	}

	// Open database
	db, err := sql.Open("sqlite3", database.DatabaseName)
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		return
	}
	defer db.Close()

	// Check if note column already exists
	var columnExists int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('task') WHERE name='note'").Scan(&columnExists)
	if err != nil {
		fmt.Printf("❌ Failed to check column existence: %v\n", err)
		return
	}

	if columnExists > 0 {
		fmt.Println("✅ Note column already exists in task table.")
		return
	}

	// Add note column
	fmt.Println("📝 Adding note column to task table...")
	_, err = db.Exec("ALTER TABLE task ADD COLUMN note TEXT")
	if err != nil {
		fmt.Printf("❌ Failed to add note column: %v\n", err)
		return
	}

	fmt.Println("✅ Successfully added note column to task table!")
	fmt.Println("🔄 Migration completed successfully!")
}

func startAPIServer() {
	fmt.Println("Projector - Project and Task Management")
	fmt.Println("======================================")
	fmt.Println()

	// Check if database exists
	if !database.DatabaseExists(database.DatabaseName) {
		fmt.Println("❌ Database not found. Please run 'projector init' first.")
		return
	}

	// Display initial tasks
	displayTasks()

	// Start API server in a goroutine
	server := api.NewServer(8080, database.DatabaseName)
	go func() {
		if err := server.Start(); err != nil {
			fmt.Printf("❌ API server error: %v\n", err)
		}
	}()

	// Wait for quit signal
	fmt.Println("🔄 API server is running. Press 'q' to quit...")

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either 'q' key or signal
	go func() {
		var input string
		fmt.Scanln(&input)
		if input == "q" {
			sigChan <- syscall.SIGINT
		}
	}()

	<-sigChan
	fmt.Println("\n👋 Shutting down Projector...")
}

func displayTasks() {
	// Get all tasks
	tasks, err := database.GetAllTasks(database.DatabaseName)
	if err != nil {
		fmt.Printf("❌ Error retrieving tasks: %v\n", err)
		return
	}

	if len(tasks) == 0 {
		fmt.Println("📝 No tasks found. Create some tasks to get started!")
		fmt.Println("Use 'projector init' to initialize the database if needed.")
		return
	}

	fmt.Printf("📋 Found %d task(s):\n\n", len(tasks))

	// Display tasks in a nice format
	for _, task := range tasks {
		fmt.Printf("  %d. %s\n", task.ID, task.Name)

		// Show note if available
		if task.Note.Valid && task.Note.String != "" {
			fmt.Printf("     📝 Note: %s\n", task.Note.String)
		}

		// Show project if available
		if task.ProjectName.Valid {
			fmt.Printf("     📁 Project: %s\n", task.ProjectName.String)
		}

		// Show due date if available
		if task.DueDate.Valid {
			fmt.Printf("     📅 Due: %s\n", task.DueDate.String)
		}

		// Show status
		fmt.Printf("     🏷️  Status: %s\n", task.StatusName)
		fmt.Println()
	}
}
