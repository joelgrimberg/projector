package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Project represents a project in the database
type Project struct {
	ID      int
	Name    string
	DueDate sql.NullString
}

// GetAllProjects retrieves all projects
func GetAllProjects(dbPath string) ([]Project, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT id, name, due_date
		FROM project
		ORDER BY id DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var project Project
		err := rows.Scan(&project.ID, &project.Name, &project.DueDate)
		if err != nil {
			return nil, err
		}
		projects = append(projects, project)
	}

	return projects, nil
}

// GetProjectByID retrieves a project by its ID
func GetProjectByID(dbPath string, projectID int) (*Project, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT id, name, due_date
		FROM project
		WHERE id = ?
	`

	var project Project
	err = db.QueryRow(query, projectID).Scan(&project.ID, &project.Name, &project.DueDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Project not found
		}
		return nil, err
	}

	return &project, nil
}

// CreateProject creates a new project in the database
func CreateProject(dbPath, name, dueDate string) (int, error) {
	// Validate input data
	if err := ValidateProjectInput(name, dueDate); err != nil {
		return 0, err
	}

	// Validate and format due date
	validatedDueDate, err := ValidateDate(dueDate)
	if err != nil {
		return 0, err
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	query := `
		INSERT INTO project (name, due_date)
		VALUES (?, ?)
	`

	result, err := db.Exec(query, name, validatedDueDate)
	if err != nil {
		return 0, err
	}

	projectID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(projectID), nil
}
