package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/joel/projector/database"
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
	http.HandleFunc("/api/actions", s.handleActions)
	http.HandleFunc("/api/projects", s.handleProjects)
	http.HandleFunc("/api/actions/", s.handleActionByID)
	http.HandleFunc("/api/projects/", s.handleProjectByID)

	// Health check endpoint
	http.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	fmt.Printf("🚀 API server starting on port %d...\n", s.port)
	fmt.Printf("📡 Endpoints available:\n")
	fmt.Printf("   GET    /api/actions      - List all actions\n")
	fmt.Printf("   PUT    /api/actions      - Create new action\n")
	fmt.Printf("   GET    /api/actions/:id  - Get action by ID\n")
	fmt.Printf("   PUT    /api/actions/:id  - Mark action as done\n")
	fmt.Printf("   DELETE /api/actions/:id  - Delete action\n")
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

// handleActions handles action-related requests
func (s *Server) handleActions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		actions, err := database.GetAllActions(s.dbPath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving actions: %v", err), http.StatusInternalServerError)
			return
		}

		// Convert to JSON response
		response := map[string]interface{}{
			"success": true,
			"count":   len(actions),
			"actions": actions,
		}

		json.NewEncoder(w).Encode(response)

	case "PUT":
		// Parse request body
		var actionRequest struct {
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

		if err := json.NewDecoder(r.Body).Decode(&actionRequest); err != nil {
			http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
			return
		}

		// Validate required fields
		if actionRequest.Name == "" {
			http.Error(w, "Action name is required", http.StatusBadRequest)
			return
		}

		if actionRequest.StatusID == 0 {
			actionRequest.StatusID = 1 // Default to 'todo' status
		}

		// Create the action
		actionID, err := database.CreateAction(s.dbPath, actionRequest.Name, actionRequest.Note, actionRequest.ProjectID, actionRequest.DueDate, actionRequest.StatusID, actionRequest.RepeatCount, actionRequest.RepeatInterval, actionRequest.RepeatPattern, actionRequest.RepeatUntil, nil)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error creating action: %v", err), http.StatusInternalServerError)
			return
		}

		// Get the created action
		action, err := database.GetActionByID(s.dbPath, actionID)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving created action: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Action created successfully",
			"action_id": actionID,
			"action":    action,
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleActionByID handles requests for a specific action
func (s *Server) handleActionByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Extract ID from URL path
	path := r.URL.Path
	if len(path) < 13 { // "/api/actions/" is 13 characters
		http.Error(w, "Invalid action ID", http.StatusBadRequest)
		return
	}

	actionIDStr := path[13:] // Remove "/api/actions/" prefix
	actionID, err := strconv.ParseUint(actionIDStr, 10, 32)
	if err != nil {
		http.Error(w, "Invalid action ID", http.StatusBadRequest)
		return
	}
	actionIDUint := uint(actionID)

	switch r.Method {
	case "GET":
		// Get action by ID
		action, err := database.GetActionByID(s.dbPath, actionIDUint)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error retrieving action: %v", err), http.StatusInternalServerError)
			return
		}

		if action == nil {
			http.Error(w, "Action not found", http.StatusNotFound)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"action":    action,
		}

		json.NewEncoder(w).Encode(response)

	case "DELETE":
		// Delete the action
		err := database.DeleteAction(s.dbPath, actionIDUint)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error deleting action: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"success": true,
			"message": "Action deleted successfully",
			"action_id": actionIDUint,
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
			// Mark action as done and handle repetition
			err := database.MarkActionAsDone(s.dbPath, actionIDUint)
			if err != nil {
				http.Error(w, fmt.Sprintf("Error marking action as done: %v", err), http.StatusInternalServerError)
				return
			}

			response := map[string]interface{}{
				"success": true,
				"message": "Action marked as done",
				"action_id": actionIDUint,
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
