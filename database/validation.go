package database

import (
	"fmt"
	"time"
)

// ValidateDate checks if a date string is valid and returns a formatted date string
func ValidateDate(dateStr string) (string, error) {
	if dateStr == "" {
		return "", nil // Empty date is valid (optional field)
	}

	// Parse the date string
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return "", fmt.Errorf("invalid date format: %s. Expected format: YYYY-MM-DD", dateStr)
	}

	// Check if the date is in the future (optional validation)
	// You can remove this if you want to allow past dates
	if date.Before(time.Now().Truncate(24 * time.Hour)) {
		return "", fmt.Errorf("date %s is in the past", dateStr)
	}

	// Return the formatted date string
	return date.Format("2006-01-02"), nil
}

// ValidateTaskInput validates task input data
func ValidateTaskInput(name string, projectID *int, dueDate string, statusID int) error {
	if name == "" {
		return fmt.Errorf("task name is required")
	}

	if len(name) > 255 {
		return fmt.Errorf("task name is too long (max 255 characters)")
	}

	if statusID <= 0 {
		return fmt.Errorf("invalid status ID")
	}

	// Validate due date if provided
	if dueDate != "" {
		_, err := ValidateDate(dueDate)
		if err != nil {
			return fmt.Errorf("due date validation failed: %v", err)
		}
	}

	return nil
}

// ValidateProjectInput validates project input data
func ValidateProjectInput(name string, dueDate string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}

	if len(name) > 255 {
		return fmt.Errorf("project name is too long (max 255 characters)")
	}

	// Validate due date if provided
	if dueDate != "" {
		_, err := ValidateDate(dueDate)
		if err != nil {
			return fmt.Errorf("due date validation failed: %v", err)
		}
	}

	return nil
}
