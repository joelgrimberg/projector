# Projector - Project and Task Management

A Go-based command-line application for managing projects and tasks with a powerful API and flexible task repetition system.

## 🚀 Quick Start

### Prerequisites
- Go 1.21 or higher
- SQLite3

### Installation & Running
```bash
# Clone the repository
git clone <your-repo-url>
cd projector

# Run the application
go run .

# Or build and run
go build -o projector .
./projector
```

## 📋 Available Commands

### Main Command
- `go run .` - Starts the application, displays all tasks, and runs the API server

### Database Commands
- `go run . init` - Initialize the database and create tables
- `go run . migrate` - Run database migrations to add new columns

## 🗄️ Database Initialization

### The `init` Command
The `init` command uses an interactive Bubble Tea TUI to set up your database:

1. **Database Creation**: Creates a new SQLite database file
2. **Table Creation**: Sets up tables for projects, tasks, statuses, and tags
3. **Schema Validation**: Ensures all required columns exist

```bash
go run . init
```

The interactive interface will guide you through:
- Database file location
- Table creation
- Schema verification

## 🌐 API Usage

The application runs an HTTP API server on port 8080. The server stays running until you press 'q' to quit.

### Base URL
```
http://localhost:8080
```

### Endpoints

#### Health Check
```bash
curl http://localhost:8080/health
```

#### Tasks

**Get All Tasks**
```bash
curl http://localhost:8080/api/tasks
```

**Get Task by ID**
```bash
curl http://localhost:8080/api/tasks/1
```

**Create New Task**
```bash
curl -X PUT http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Task Name",
    "note": "Optional note",
    "project_id": 1,
    "due_date": "2024-12-31",
    "status_id": 1,
    "repeat_count": 5,
    "repeat_interval": "week",
    "repeat_pattern": "mon,tue,wed,thu,fri",
    "repeat_until": "2025-06-30"
  }'
```

**Mark Task as Done**
```bash
curl -X PUT http://localhost:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"action": "done"}'
```

**Delete Task**
```bash
curl -X DELETE http://localhost:8080/api/tasks/1
```

#### Projects

**Get All Projects**
```bash
curl http://localhost:8080/api/projects
```

**Get Project by ID**
```bash
curl http://localhost:8080/api/projects/1
```

**Delete Project**
```bash
curl -X DELETE http://localhost:8080/api/projects/1
```

**Create New Project**
```bash
curl -X PUT http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Project Name",
    "due_date": "2024-12-31"
  }'
```

### API Response Format
```json
{
  "success": true,
  "message": "Operation completed",
  "data": {
    "id": 1,
    "name": "Task Name",
    "note": "Optional note",
    "due_date": "2024-12-31",
    "status": "todo",
    "project": "Project Name"
  }
}
```

## 🔄 Task Repetition System

The application supports flexible task repetition with custom patterns and intervals.

### Basic Repetition
```bash
curl -X PUT http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Check-in",
    "repeat_count": 30,
    "repeat_interval": "day"
  }'
```

### Weekly Patterns

**Weekdays Only (Monday-Friday)**
```bash
curl -X PUT http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Daily Standup",
    "repeat_count": 20,
    "repeat_interval": "week",
    "repeat_pattern": "mon,tue,wed,thu,fri"
  }'
```

**Specific Days**
```bash
curl -X PUT http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Weekend Planning",
    "repeat_count": 8,
    "repeat_interval": "week",
    "repeat_pattern": "fri,sat"
  }'
```

**Single Day**
```bash
curl -X PUT http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Monday Review",
    "repeat_count": 12,
    "repeat_interval": "week",
    "repeat_pattern": "monday"
  }'
```

### Supported Intervals
- `minute` - Every minute
- `hour` - Every hour
- `day` - Every day
- `week` - Every week (with custom patterns)
- `month` - Every month
- `year` - Every year

### Weekly Pattern Formats

**Full Names**
- `monday,tuesday,wednesday,thursday,friday`
- `monday,wednesday,friday`

**Abbreviations**
- `mon,tue,wed,thu,fri`
- `mon,wed,fri`

**Single Letters**
- `m,t,w,r,f` (Monday, Tuesday, Wednesday, Thursday, Friday)
- `m,w,f` (Monday, Wednesday, Friday)

### How Repetition Works

1. **Task Created** with repetition settings
2. **Mark as Done** → Next task automatically created
3. **Pattern Calculation** → Next due date calculated based on interval and pattern
4. **Repeat Count** → Decrements with each completion
5. **Repeat Until** → Optional end date for repetition

### Example Workflow

**Create a weekday task:**
```bash
curl -X PUT http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Team Meeting",
    "repeat_count": 10,
    "repeat_interval": "week",
    "repeat_pattern": "mon,wed,fri"
  }'
```

**Task progression:**
- Monday 2024-12-30 → Mark done → Tuesday 2024-12-31 created
- Tuesday 2024-12-31 → Mark done → Wednesday 2025-01-01 created
- Wednesday 2025-01-01 → Mark done → Friday 2025-01-03 created
- Friday 2025-01-03 → Mark done → Monday 2025-01-06 created (next week)

## 🗃️ Database Schema

### Tables
- **project**: Projects with names and due dates
- **task**: Tasks with names, notes, due dates, status, and repetition settings
- **status**: Task statuses (todo, in_progress, done)
- **tag**: Task tags for categorization
- **task_tag**: Many-to-many relationship between tasks and tags

### Task Repetition Fields
- `repeat_count`: Number of repetitions remaining
- `repeat_interval`: Time interval (minute, hour, day, week, month, year)
- `repeat_pattern`: Custom pattern for weekly repetition
- `repeat_until`: Optional end date for repetition
- `parent_task_id`: Links to the original task for tracking

## 🔧 Development

### Project Structure
```
projector/
├── main.go              # Main entry point and CLI commands
├── database/
│   ├── database.go      # Database connection and table creation
│   └── tasks.go         # Task-related database operations
├── api/
│   └── server.go        # HTTP API server
├── models/
│   └── result.go        # Data models
├── ui/
│   └── ui.go           # Bubble Tea TUI for init command
├── go.mod               # Go module dependencies
└── go.sum               # Dependency checksums
```

### Adding New Features
1. **Database Changes**: Update schema in `database/database.go`
2. **API Endpoints**: Add handlers in `api/server.go`
3. **CLI Commands**: Extend `main.go` with new Cobra commands
4. **Migrations**: Add new columns to the migration system

### Running Tests
```bash
go test ./...
```

## 📝 Examples

### Complete Workflow Example

1. **Initialize Database**
```bash
go run . init
```

2. **Start Application**
```bash
go run .  # Shows tasks and starts API server
```

3. **Create a Project**
```bash
curl -X PUT http://localhost:8080/api/projects \
  -H "Content-Type: application/json" \
  -d '{"name": "Website Redesign", "due_date": "2025-03-31"}'
```

4. **Create a Repeating Task**
```bash
curl -X PUT http://localhost:8080/api/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Code Review",
    "note": "Review pull requests",
    "project_id": 1,
    "due_date": "2024-12-30",
    "status_id": 1,
    "repeat_count": 15,
    "repeat_interval": "week",
    "repeat_pattern": "tue,thu"
  }'
```

5. **Mark Tasks as Done**
```bash
curl -X PUT http://localhost:8080/api/tasks/1 \
  -H "Content-Type: application/json" \
  -d '{"action": "done"}'
```

6. **View All Tasks**
```bash
curl http://localhost:8080/api/tasks | jq '.tasks[] | {name, due_date, repeat_pattern}'
```

## 🚨 Troubleshooting

### Common Issues

**Database Locked**
- Ensure no other processes are using the database
- Check file permissions

**Port Already in Use**
- Change the port in `api/server.go` if 8080 is occupied
- Kill processes using the port: `lsof -ti:8080 | xargs kill -9`

**Migration Errors**
- Run `go run . migrate` to add missing columns
- Check database schema with SQLite browser

### Getting Help
- Check the application logs for error messages
- Verify database file exists and is accessible
- Ensure all required Go dependencies are installed

## 📄 License

[Add your license information here]

---

**Happy Projecting! 🎯**
