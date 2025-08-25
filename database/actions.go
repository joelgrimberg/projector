package database

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Action represents an action in the database
type Action struct {
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
	ParentActionID sql.NullInt64
	ProjectName    sql.NullString
	StatusName     string
}

// GetAllActions retrieves all actions with their project and status information
func GetAllActions(dbPath string) ([]Action, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT 
			a.id, 
			a.project_id, 
			a.name, 
			a.note,
			a.due_date, 
			a.status_id,
			a.repeat_count,
			a.repeat_interval,
			a.repeat_pattern,
			a.repeat_until,
			a.parent_action_id,
			p.name as project_name,
			s.name as status_name
		FROM action a
		LEFT JOIN project p ON a.project_id = p.id
		LEFT JOIN status s ON a.status_id = s.id
		ORDER BY a.id DESC
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []Action
	for rows.Next() {
		var action Action
		err := rows.Scan(
			&action.ID,
			&action.ProjectID,
			&action.Name,
			&action.Note,
			&action.DueDate,
			&action.StatusID,
			&action.RepeatCount,
			&action.RepeatInterval,
			&action.RepeatPattern,
			&action.RepeatUntil,
			&action.ParentActionID,
			&action.ProjectName,
			&action.StatusName,
		)
		if err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}

	return actions, nil
}

// GetActionByID retrieves an action by its ID
func GetActionByID(dbPath string, actionID uint) (*Action, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := `
		SELECT 
			a.id, 
			a.project_id, 
			a.name, 
			a.note,
			a.due_date, 
			a.status_id,
			a.repeat_count,
			a.repeat_interval,
			a.repeat_pattern,
			a.repeat_until,
			a.parent_action_id,
			p.name as project_name,
			s.name as status_name
		FROM action a
		LEFT JOIN project p ON a.project_id = p.id
		LEFT JOIN status s ON a.status_id = s.id
		WHERE a.id = ?
	`

	var action Action
	err = db.QueryRow(query, actionID).Scan(
		&action.ID,
		&action.ProjectID,
		&action.Name,
		&action.Note,
		&action.DueDate,
		&action.StatusID,
		&action.RepeatCount,
		&action.RepeatInterval,
		&action.RepeatPattern,
		&action.RepeatUntil,
		&action.ParentActionID,
		&action.ProjectName,
		&action.StatusName,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Action not found
		}
		return nil, err
	}

	return &action, nil
}

// CreateAction creates a new action in the database
func CreateAction(dbPath, name, note string, projectID *uint, dueDate string, statusID uint, repeatCount uint, repeatInterval, repeatPattern, repeatUntil string, parentActionID *uint) (uint, error) {
	// Validate input data
	if err := ValidateActionInput(name, projectID, dueDate, statusID); err != nil {
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
		INSERT INTO action (name, note, project_id, due_date, status_id, repeat_count, repeat_interval, repeat_pattern, repeat_until, parent_action_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var result sql.Result
	if projectID != nil {
		result, err = db.Exec(query, name, note, *projectID, validatedDueDate, statusID, repeatCount, repeatInterval, repeatPattern, repeatUntil, parentActionID)
	} else {
		result, err = db.Exec(query, name, note, nil, validatedDueDate, statusID, repeatCount, repeatInterval, repeatPattern, repeatUntil, parentActionID)
	}

	if err != nil {
		return 0, err
	}

	actionID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return uint(actionID), nil
}

// CreateNextRepeatedAction creates the next occurrence of a repeating action
func CreateNextRepeatedAction(dbPath string, originalAction *Action) (uint, error) {
	if originalAction.RepeatCount <= 0 || originalAction.RepeatInterval.String == "" {
		return 0, fmt.Errorf("action is not configured for repetition")
	}

	// Calculate next due date based on interval
	nextDueDate, err := calculateNextDueDate(originalAction.DueDate.String, originalAction.RepeatInterval.String, originalAction.RepeatPattern.String)
	if err != nil {
		return 0, err
	}

	// Check if we've reached the repeat until date
	if originalAction.RepeatUntil.Valid && originalAction.RepeatUntil.String != "" {
		untilDate, err := time.Parse("2006-01-02", originalAction.RepeatUntil.String)
		if err == nil && nextDueDate.After(untilDate) {
			return 0, fmt.Errorf("repetition limit reached")
		}
	}

	// Create the next action
	var projectID *uint
	if originalAction.ProjectID.Valid {
		projectIDUint := uint(originalAction.ProjectID.Int64)
		projectID = &projectIDUint
	}

	nextActionID, err := CreateAction(
		dbPath,
		originalAction.Name,
		originalAction.Note.String,
		projectID,
		nextDueDate.Format("2006-01-02"),
		originalAction.StatusID,
		originalAction.RepeatCount-1, // Decrease repeat count
		originalAction.RepeatInterval.String,
		originalAction.RepeatPattern.String,
		originalAction.RepeatUntil.String,
		&originalAction.ID, // Set this as the parent action
	)

	if err != nil {
		return 0, err
	}

	return nextActionID, nil
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

// MarkActionAsDone marks an action as done and creates the next repeated action if configured
func MarkActionAsDone(dbPath string, actionID uint) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Get the action details
	action, err := GetActionByID(dbPath, actionID)
	if err != nil {
		return err
	}
	if action == nil {
		return fmt.Errorf("action not found")
	}

	// Update status to done (assuming status ID 2 is 'done')
	_, err = db.Exec("UPDATE action SET status_id = 2 WHERE id = ?", actionID)
	if err != nil {
		return err
	}

	// If action has repetition configured, create the next occurrence
	if action.RepeatCount > 0 && action.RepeatInterval.Valid {
		_, err = CreateNextRepeatedAction(dbPath, action)
		if err != nil {
			// Log the error but don't fail the operation
			fmt.Printf("Warning: Failed to create next repeated action: %v\n", err)
		}
	}

	return nil
}

// DeleteAction deletes an action from the database
func DeleteAction(dbPath string, actionID uint) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// Check if action exists
	action, err := GetActionByID(dbPath, actionID)
	if err != nil {
		return fmt.Errorf("error checking action existence: %v", err)
	}
	if action == nil {
		return fmt.Errorf("action not found")
	}

	// Delete the action
	query := "DELETE FROM action WHERE id = ?"
	_, err = db.Exec(query, actionID)
	if err != nil {
		return fmt.Errorf("failed to delete action: %v", err)
	}

	return nil
}
