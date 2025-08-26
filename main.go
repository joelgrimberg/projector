package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joelgrimberg/projector/api"
	"github.com/joelgrimberg/projector/database"
	"github.com/joelgrimberg/projector/ui"

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
			verbose, _ := cmd.Flags().GetBool("verbose")
			startAPIServer(verbose)
		},
	}

	// Add verbose flag
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")

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
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate database schema to add note and repeat fields to actions",
		Run: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			runMigration(verbose)
		},
	}
	
	// Add verbose flag to migrate command
	cmd.Flags().BoolP("verbose", "v", false, "Enable verbose output")
	return cmd
}

func runMigration(verbose bool) {
	if verbose {
		fmt.Println("🔄 Starting database migration...")
	}

	// Check if database exists
	if !database.DatabaseExists(database.GetDatabasePath()) {
		fmt.Println("❌ Database not found. Please run 'projector init' first.")
		return
	}

	// Open database
	db, err := sql.Open("sqlite3", database.GetDatabasePath())
	if err != nil {
		fmt.Printf("❌ Failed to open database: %v\n", err)
		return
	}
	defer db.Close()

	// First, check if we need to rename the task table to action table
	var tableExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task'").Scan(&tableExists)
	if err != nil {
		fmt.Printf("❌ Error checking for task table: %v\n", err)
		return
	}

	if tableExists > 0 {
		if verbose {
			fmt.Println("🔄 Renaming 'task' table to 'action' table...")
		}
		
		// Rename the task table to action table
		_, err = db.Exec("ALTER TABLE task RENAME TO action")
		if err != nil {
			fmt.Printf("❌ Failed to rename task table: %v\n", err)
			return
		}
		if verbose {
			fmt.Println("✅ Table renamed successfully")
		}

		// Rename the task_tag table to action_tag table
		err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='task_tag'").Scan(&tableExists)
		if err == nil && tableExists > 0 {
			if verbose {
				fmt.Println("🔄 Renaming 'task_tag' table to 'action_tag' table...")
			}
			_, err = db.Exec("ALTER TABLE task_tag RENAME TO action_tag")
			if err != nil {
				fmt.Printf("❌ Failed to rename task_tag table: %v\n", err)
				return
			}
			if verbose {
				fmt.Println("✅ task_tag table renamed successfully")
			}
			
			// Rename the task_id column to action_id in the action_tag table
			if verbose {
				fmt.Println("🔄 Renaming 'task_id' column to 'action_id' in action_tag table...")
			}
			_, err = db.Exec("ALTER TABLE action_tag RENAME COLUMN task_id TO action_id")
			if err != nil {
				fmt.Printf("❌ Failed to rename task_id column: %v\n", err)
				return
			}
			if verbose {
				fmt.Println("✅ Column renamed successfully")
			}
		}

		// Rename the parent_task_id column to parent_action_id
		if verbose {
			fmt.Println("🔄 Renaming 'parent_task_id' column to 'parent_action_id'...")
		}
		_, err = db.Exec("ALTER TABLE action RENAME COLUMN parent_task_id TO parent_action_id")
		if err != nil {
			fmt.Printf("❌ Failed to rename parent_task_id column: %v\n", err)
			return
		}
		if verbose {
			fmt.Println("✅ Column renamed successfully")
		}
	}

	// Always check and fix the action_tag table column names if needed
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='action_tag'").Scan(&tableExists)
	if err == nil && tableExists > 0 {
		// Check if the action_tag table still has the old task_id column
		var columnExists int
		err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('action_tag') WHERE name='task_id'").Scan(&columnExists)
		if err == nil && columnExists > 0 {
			if verbose {
				fmt.Println("🔄 Fixing 'task_id' column name to 'action_id' in action_tag table...")
			}
			_, err = db.Exec("ALTER TABLE action_tag RENAME COLUMN task_id TO action_id")
			if err != nil {
				fmt.Printf("❌ Failed to rename task_id column: %v\n", err)
			} else {
				if verbose {
					fmt.Println("✅ Column renamed successfully")
				}
			}
		}
	}

	// List of columns to add (these will be skipped if they already exist)
	columns := []struct {
		name    string
		sql     string
		display string
	}{
		{"note", "ALTER TABLE action ADD COLUMN note TEXT", "note"},
		{"repeat_count", "ALTER TABLE action ADD COLUMN repeat_count INTEGER DEFAULT 0", "repeat_count"},
		{"repeat_interval", "ALTER TABLE action ADD COLUMN repeat_interval TEXT", "repeat_interval"},
		{"repeat_pattern", "ALTER TABLE action ADD COLUMN repeat_pattern TEXT", "repeat_pattern"},
		{"repeat_until", "ALTER TABLE action ADD COLUMN repeat_until DATE", "repeat_until"},
		{"parent_action_id", "ALTER TABLE action ADD COLUMN parent_action_id INTEGER", "parent_action_id"},
	}

	// Add missing columns
	for _, column := range columns {
		// Check if column already exists
		var columnExists int
		err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('action') WHERE name='%s'", column.name)).Scan(&columnExists)
		if err != nil {
			fmt.Printf("⚠️ Could not check if column '%s' exists: %v\n", column.name, err)
			continue
		}

		if columnExists == 0 {
			if verbose {
				fmt.Printf("📝 Adding %s column to action table...\n", column.display)
			}
			_, err = db.Exec(column.sql)
			if err != nil {
				fmt.Printf("❌ Failed to add %s column: %v\n", column.display, err)
				continue
			}
			if verbose {
				fmt.Printf("✅ Successfully added %s column\n", column.display)
			}
		} else {
			if verbose {
				fmt.Printf("✅ %s column already exists\n", column.display)
			}
		}
	}

	if verbose {
		fmt.Println("🔄 Migration completed successfully!")
	}
}

func startAPIServer(verbose bool) {
	fmt.Println("Projector - Project and Action Management")
	fmt.Println("======================================")
	fmt.Println()

	// Check if database exists
	if !database.DatabaseExists(database.GetDatabasePath()) {
		fmt.Println("❌ Database not found. Please run 'projector init' first.")
		return
	}

	// Run migration to ensure database schema is up to date
	if verbose {
		fmt.Println("🔄 Checking database schema...")
	}
	runMigration(verbose)

	// Display initial actions
	displayActions()

	// Start API server in a goroutine
	server := api.NewServer(8080, database.GetDatabasePath())
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

func displayActions() {
	// Get all actions
	actions, err := database.GetAllActions(database.GetDatabasePath())
	if err != nil {
		fmt.Printf("❌ Error retrieving actions: %v\n", err)
		return
	}

	if len(actions) == 0 {
		fmt.Println("📝 No actions found. Create some actions to get started!")
		return
	}

	fmt.Printf("📋 Found %d action(s):\n\n", len(actions))

	// Display actions in a nice format
	for _, action := range actions {
		fmt.Printf("  %d. %s\n", action.ID, action.Name)

		// Show note if available
		if action.Note.Valid && action.Note.String != "" {
			fmt.Printf("     📝 Note: %s\n", action.Note.String)
		}

		// Show project if available
		if action.ProjectName.Valid {
			fmt.Printf("     📁 Project: %s\n", action.ProjectName.String)
		}

		// Show due date if available
		if action.DueDate.Valid {
			fmt.Printf("     📅 Due: %s\n", action.DueDate.String)
		}

		// Show repeat information if available
		if action.RepeatCount > 0 && action.RepeatInterval.Valid {
			fmt.Printf("     🔄 Repeat: %d times every %s", action.RepeatCount, action.RepeatInterval.String)
			if action.RepeatPattern.Valid && action.RepeatPattern.String != "" {
				fmt.Printf(" on %s", action.RepeatPattern.String)
			}
			if action.RepeatUntil.Valid {
				fmt.Printf(" until %s", action.RepeatUntil.String)
			}
			fmt.Println()
		}

		// Show status
		fmt.Printf("     🏷️  Status: %s\n", action.StatusName)
		fmt.Println()
	}
}
