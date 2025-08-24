package database

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Task represents a task in the database
type Task struct {
	ID             uint
	ProjectID      sql.NullInt64
	Name           string
	Note           sql.NullString
	DueDate        sql.NullString
	StatusID       uint
	RepeatCount    uint
	RepeatInterval sql.NullString
	RepeatPattern  sql.NullString
	RepeatUntil    sql.NullString
	ParentTaskID   sql.NullInt64
	ProjectName    sql.NullString
	StatusName     string
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
			t.repeat_count,
			t.repeat_interval,
			t.repeat_pattern,
			t.repeat_until,
			t.parent_task_id,
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
			&task.RepeatCount,
			&task.RepeatInterval,
			&task.RepeatPattern,
			&task.RepeatUntil,
			&task.ParentTaskID,
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
func GetTaskByID(dbPath string, taskID uint) (*Task, error) {
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
			t.repeat_count,
			t.repeat_interval,
			t.repeat_pattern,
			t.repeat_until,
			t.parent_task_id,
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
		&task.RepeatCount,
		&task.RepeatInterval,
		&task.RepeatPattern,
		&task.RepeatUntil,
		&task.ParentTaskID,
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
func CreateTask(dbPath, name, note string, projectID *uint, dueDate string, statusID uint, repeatCount uint, repeatInterval, repeatPattern, repeatUntil string, parentTaskID *uint) (uint, error) {
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
		INSERT INTO task (name, note, project_id, due_date, status_id, repeat_count, repeat_interval, repeat_pattern, repeat_until, parent_task_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var result sql.Result
	if projectID != nil {
		result, err = db.Exec(query, name, note, *projectID, validatedDueDate, statusID, repeatCount, repeatInterval, repeatPattern, repeatUntil, parentTaskID)
	} else {
		result, err = db.Exec(query, name, note, nil, validatedDueDate, statusID, repeatCount, repeatInterval, repeatPattern, repeatUntil, parentTaskID)
	}

	if err != nil {
		return 0, err
	}

	taskID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint(taskID), nil
}

// CreateNextRepeatedTask creates the next occurrence of a repeating task
func CreateNextRepeatedTask(dbPath string, originalTask *Task) (uint, error) {
	if originalTask.RepeatCount <= 0 || originalTask.RepeatInterval.String == "" {
		return 0, fmt.Errorf("task is not configured for repetition")
	}

	// Calculate next due date based on interval
	nextDueDate, err := calculateNextDueDate(originalTask.DueDate.String, originalTask.RepeatInterval.String, originalTask.RepeatPattern.String)
	if err != nil {
		return 0, err
	}

	// Check if we've reached the repeat until date
	if originalTask.RepeatUntil.Valid && originalTask.RepeatUntil.String != "" {
		untilDate, err := time.Parse("2006-01-02", originalTask.RepeatUntil.String)
		if err == nil && nextDueDate.After(untilDate) {
			return 0, fmt.Errorf("repetition limit reached")
		}
	}

	// Create the next task
	var projectID *uint
	if originalTask.ProjectID.Valid {
		projectIDUint := uint(originalTask.ProjectID.Int64)
		projectID = &projectIDUint
	}

	nextTaskID, err := CreateTask(
		dbPath,
		originalTask.Name,
		originalTask.Note.String,
		projectID,
		nextDueDate.Format("2006-01-02"),
		originalTask.StatusID,
		originalTask.RepeatCount-1, // Decrease repeat count
		originalTask.RepeatInterval.String,
		originalTask.RepeatPattern.String,
		originalTask.RepeatUntil.String,
		&originalTask.ID, // Set this as the parent task
	)

	if err != nil {
		return 0, err
	}

	return nextTaskID, nil
}

// calculateNextDueDate calculates the next due date based on the interval and pattern
func calculateNextDueDate(currentDueDate, interval, pattern string) (time.Time, error) {
	if currentDueDate == "" {
		return time.Now(), fmt.Errorf("no current due date")
	}

	date, err := time.Parse("2006-01-02", currentDueDate)
	if err != nil {
		return time.Time{}, err
	}

	switch interval {
	case "minute":
		return date.Add(time.Minute), nil
	case "hour":
		return date.Add(time.Hour), nil
	case "day":
		return date.AddDate(0, 0, 1), nil
	case "week":
		return calculateNextWeeklyDate(date, pattern)
	case "month":
		return date.AddDate(0, 1, 0), nil
	case "year":
		return date.AddDate(1, 0, 0), nil
	default:
		return time.Time{}, fmt.Errorf("invalid interval: %s", interval)
	}
}

// calculateNextWeeklyDate calculates the next weekly date based on the pattern
func calculateNextWeeklyDate(currentDate time.Time, pattern string) (time.Time, error) {
	if pattern == "" {
		// Default: every week on the same day
		return currentDate.AddDate(0, 0, 7), nil
	}

	// Parse pattern like "mon,tue,wed,thu,fri" or "monday,tuesday,wednesday,thursday,friday"
	days := parseWeeklyPattern(pattern)
	if len(days) == 0 {
		return currentDate.AddDate(0, 0, 7), nil
	}

	// Find the next occurrence
	currentWeekday := int(currentDate.Weekday())

	// Look for the next day in the current week
	for _, day := range days {
		if day > currentWeekday {
			daysToAdd := day - currentWeekday
			return currentDate.AddDate(0, 0, daysToAdd), nil
		}
	}

	// If no more days this week, go to next week and find the first day
	nextWeek := currentDate.AddDate(0, 0, 7)
	firstDay := days[0]
	currentWeekday = int(nextWeek.Weekday())
	daysToAdd := firstDay - currentWeekday
	if daysToAdd < 0 {
		daysToAdd += 7
	}
	return nextWeek.AddDate(0, 0, daysToAdd), nil
}

// parseWeeklyPattern parses weekly pattern string into weekday numbers
func parseWeeklyPattern(pattern string) []int {
	var days []int
	parts := strings.Split(strings.ToLower(pattern), ",")

	weekdayMap := map[string]int{
		"monday": 1, "mon": 1, "m": 1,
		"tuesday": 2, "tue": 2, "tu": 2, "t": 2,
		"wednesday": 3, "wed": 3, "w": 3,
		"thursday": 4, "thu": 4, "th": 4, "r": 4,
		"friday": 5, "fri": 5, "f": 5,
		"saturday": 6, "sat": 6, "sa": 6, "s": 6,
		"sunday": 0, "sun": 0, "su": 0, "u": 0,
	}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if dayNum, exists := weekdayMap[part]; exists {
			days = append(days, dayNum)
		}
	}

	// Sort days for consistent ordering
	sort.Ints(days)
	return days
}

// MarkTaskAsDone marks a task as done and creates the next repeated task if configured
func MarkTaskAsDone(dbPath string, taskID uint) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get the task details
	task, err := GetTaskByID(dbPath, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}

	// Update status to done (assuming status ID 2 is 'done')
	_, err = db.Exec("UPDATE task SET status_id = 2 WHERE id = ?", taskID)
	if err != nil {
		return err
	}

	// If task has repetition configured, create the next occurrence
	if task.RepeatCount > 0 && task.RepeatInterval.Valid {
		_, err = CreateNextRepeatedTask(dbPath, task)
		if err != nil {
			// Log the error but don't fail the operation
			fmt.Printf("Warning: Failed to create next repeated task: %v\n", err)
		}
	}

	return nil
}

// DeleteTask deletes a task from the database
func DeleteTask(dbPath string, taskID uint) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Check if task exists
	task, err := GetTaskByID(dbPath, taskID)
	if err != nil {
		return fmt.Errorf("error checking task existence: %v", err)
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}

	// Delete the task
	query := "DELETE FROM task WHERE id = ?"
	_, err = db.Exec(query, taskID)
	if err != nil {
		return fmt.Errorf("failed to delete task: %v", err)
	}

	return nil
}
