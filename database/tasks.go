package database

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

// Task represents a task in the database
type Task struct {
	ID          int
	ProjectID   sql.NullInt64
	Name        string
	Note        sql.NullString
	DueDate     sql.NullString
	StatusID    int
	ProjectName sql.NullString
	StatusName  string
}

// GetAllTasks retrieves all tasks with their project and status information
func GetAllTasks(dbPath string) ([]Task, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT 
			t.id, 
			t.project_id, 
			t.name, 
			t.note,
			t.due_date, 
			t.status_id,
			p.name as project_name,
			s.name as status_name
		FROM task t
		LEFT JOIN project p ON t.project_id = p.id
		LEFT JOIN status s ON t.status_id = s.id
		ORDER BY t.id DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&task.Name,
			&task.Note,
			&task.DueDate,
			&task.StatusID,
			&task.ProjectName,
			&task.StatusName,
		)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// GetTaskByID retrieves a task by its ID
func GetTaskByID(dbPath string, taskID int) (*Task, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT 
			t.id, 
			t.project_id, 
			t.name, 
			t.note,
			t.due_date, 
			t.status_id,
			p.name as project_name,
			s.name as status_name
		FROM task t
		LEFT JOIN project p ON t.project_id = p.id
		LEFT JOIN status s ON t.status_id = s.id
		WHERE t.id = ?
	`

	var task Task
	err = db.QueryRow(query, taskID).Scan(
		&task.ID,
		&task.ProjectID,
		&task.Name,
		&task.Note,
		&task.DueDate,
		&task.StatusID,
		&task.ProjectName,
		&task.StatusName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Task not found
		}
		return nil, err
	}

	return &task, nil
}

// CreateTask creates a new task in the database
func CreateTask(dbPath, name, note string, projectID *int, dueDate string, statusID int) (int, error) {
	// Validate input data
	if err := ValidateTaskInput(name, projectID, dueDate, statusID); err != nil {
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
		INSERT INTO task (name, note, project_id, due_date, status_id)
		VALUES (?, ?, ?, ?, ?)
	`

	var result sql.Result
	if projectID != nil {
		result, err = db.Exec(query, name, note, *projectID, validatedDueDate, statusID)
	} else {
		result, err = db.Exec(query, name, note, nil, validatedDueDate, statusID)
	}

	if err != nil {
		return 0, err
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(taskID), nil
}
