package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"main/database"
)

// Server represents the HTTP API server
type Server struct {
	port   int
	dbPath string
}

// NewServer creates a new API server
func NewServer(port int, dbPath string) *Server {
	return &Server{
		port:   port,
		dbPath: dbPath,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Set up routes
	http.HandleFunc("/api/tasks", s.handleTasks)
	http.HandleFunc("/api/projects", s.handleProjects)
	http.HandleFunc("/api/tasks/", s.handleTaskByID)
	http.HandleFunc("/api/projects/", s.handleProjectByID)

	// Health check endpoint
	http.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("🚀 API server starting on port %d...\n", s.port)
	fmt.Printf("📡 Endpoints available:\n")
	fmt.Printf("   GET    /api/tasks      - List all tasks\n")
	fmt.Printf("   PUT    /api/tasks      - Create new task\n")
	fmt.Printf("   GET    /api/tasks/:id  - Get task by ID\n")
	fmt.Printf("   PUT    /api/tasks/:id  - Mark task as done\n")
	fmt.Printf("   DELETE /api/tasks/:id  - Delete task\n")
	fmt.Printf("   GET    /api/projects   - List all projects\n")
	fmt.Printf("   PUT    /api/projects   - Create new project\n")
	fmt.Printf("   GET    /api/projects/:id - Get project by ID\n")
	fmt.Printf("   DELETE /api/projects/:id - Delete project\n")
	fmt.Printf("   GET    /health         - Health check\n")
	fmt.Printf("   Press 'q' to quit\n\n")

	return http.ListenAndServe(addr, nil)
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"message": "Projector API is running",
	})
}

// handleTasks handles task-related requests
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		tasks, err := database.GetAllTasks(s.dbPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving tasks: %v", err), http.StatusInternalServerError)
			return
		}

		// Convert to JSON response
		response := map[string]interface{}{
			"success": true,
			"count":   len(tasks),
			"tasks":   tasks,
		}

		json.NewEncoder(w).Encode(response)

	case "PUT":
		// Parse request body
		var taskRequest struct {
			Name           string `json:"name"`
			Note           string `json:"note,omitempty"`
			ProjectID      *uint  `json:"project_id,omitempty"`
			DueDate        string `json:"due_date,omitempty"`
			StatusID       uint   `json:"status_id"`
			RepeatCount    uint   `json:"repeat_count,omitempty"`
			RepeatInterval string `json:"repeat_interval,omitempty"`
			RepeatPattern  string `json:"repeat_pattern,omitempty"`
			RepeatUntil    string `json:"repeat_until,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&taskRequest); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		// Validate required fields
		if taskRequest.Name == "" {
			http.Error(w, "Task name is required", http.StatusBadRequest)
			return
		}

		if taskRequest.StatusID == 0 {
			taskRequest.StatusID = 1 // Default to 'todo' status
		}

		// Create the task
		taskID, err := database.CreateTask(s.dbPath, taskRequest.Name, taskRequest.Note, taskRequest.ProjectID, taskRequest.DueDate, taskRequest.StatusID, taskRequest.RepeatCount, taskRequest.RepeatInterval, taskRequest.RepeatPattern, taskRequest.RepeatUntil, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creating task: %v", err), http.StatusInternalServerError)
			return
		}

		// Get the created task
		task, err := database.GetTaskByID(s.dbPath, taskID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving created task: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Task created successfully",
			"task_id": taskID,
			"task":    task,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTaskByID handles requests for a specific task
func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from URL path
	path := r.URL.Path
	if len(path) < 12 { // "/api/tasks/" is 12 characters
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}

	taskIDStr := path[12:] // Remove "/api/tasks/" prefix
	taskID, err := strconv.ParseUint(taskIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid task ID", http.StatusBadRequest)
		return
	}
	taskIDUint := uint(taskID)

	switch r.Method {
	case "GET":
		// Get task by ID
		task, err := database.GetTaskByID(s.dbPath, taskIDUint)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving task: %v", err), http.StatusInternalServerError)
			return
		}

		if task == nil {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"task":    task,
		}

		json.NewEncoder(w).Encode(response)

	case "DELETE":
		// Delete the task
		err := database.DeleteTask(s.dbPath, taskIDUint)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error deleting task: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Task deleted successfully",
			"task_id": taskIDUint,
		}

		json.NewEncoder(w).Encode(response)

	case "PUT":
		// Parse request body for action
		var actionRequest struct {
			Action string `json:"action"`
		}

		if err := json.NewDecoder(r.Body).Decode(&actionRequest); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		switch actionRequest.Action {
		case "done":
			// Mark task as done and handle repetition
			err := database.MarkTaskAsDone(s.dbPath, taskIDUint)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error marking task as done: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"success": true,
				"message": "Task marked as done",
				"task_id": taskIDUint,
			}

			json.NewEncoder(w).Encode(response)

		default:
			http.Error(w, fmt.Sprintf("Unknown action: %s", actionRequest.Action), http.StatusBadRequest)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProjects handles project-related requests
func (s *Server) handleProjects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		projects, err := database.GetAllProjects(s.dbPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving projects: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success":  true,
			"count":    len(projects),
			"projects": projects,
		}

		json.NewEncoder(w).Encode(response)

	case "PUT":
		// Parse request body
		var projectRequest struct {
			Name    string `json:"name"`
			DueDate string `json:"due_date,omitempty"`
		}

		if err := json.NewDecoder(r.Body).Decode(&projectRequest); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		// Validate required fields
		if projectRequest.Name == "" {
			http.Error(w, "Project name is required", http.StatusBadRequest)
			return
		}

		// Create the project
		projectID, err := database.CreateProject(s.dbPath, projectRequest.Name, projectRequest.DueDate)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creating project: %v", err), http.StatusInternalServerError)
			return
		}

		// Get the created project
		project, err := database.GetProjectByID(s.dbPath, projectID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving created project: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success":    true,
			"message":    "Project created successfully",
			"project_id": projectID,
			"project":    project,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProjectByID handles requests for a specific project
func (s *Server) handleProjectByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from URL path
	path := r.URL.Path
	if len(path) < 15 { // "/api/projects/" is 15 characters
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	projectIDStr := path[15:] // Remove "/api/projects/" prefix
	projectID, err := strconv.ParseUint(projectIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}
	projectIDUint := uint(projectID)

	switch r.Method {
	case "GET":
		// Get project by ID
		project, err := database.GetProjectByID(s.dbPath, projectIDUint)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving project: %v", err), http.StatusInternalServerError)
			return
		}

		if project == nil {
			http.Error(w, "Project not found", http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"project": project,
		}

		json.NewEncoder(w).Encode(response)

	case "DELETE":
		// Delete the project
		err := database.DeleteProject(s.dbPath, projectIDUint)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error deleting project: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success":    true,
			"message":    "Project deleted successfully",
			"project_id": projectIDUint,
		}

		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
}
