# Projector

A CLI application for project and action management with a modern terminal UI.

## Features

- 📋 **Action Management**: Create, track, and manage your daily actions
- 📁 **Project Organization**: Group actions by projects
- 🔄 **Repeating Actions**: Set up recurring tasks with custom intervals
- 🏷️ **Status Tracking**: Mark actions as todo or done
- 📝 **Notes & Due Dates**: Add context and deadlines to your actions
- 🌐 **REST API**: Full HTTP API for integration with other tools
- 🎨 **Beautiful UI**: Modern terminal interface built with Bubble Tea

## Installation

### Via Homebrew (Recommended)
```bash
brew install joelgrimberg/tap/projector
```

### From Source
```bash
git clone https://github.com/joelgrimberg/projector.git
cd projector
go install
```

## Quick Start

1. **Initialize the database**:
   ```bash
   projector init
   ```

2. **Start the application**:
   ```bash
   projector
   ```

3. **Create your first action**:
   ```bash
   curl -X PUT http://localhost:8080/api/actions \
     -H "Content-Type: application/json" \
     -d '{"name": "Write documentation", "status_id": 1}'
   ```

## Usage

### Commands

- `projector` - Start the API server and display actions
- `projector init` - Initialize the database and create tables
- `projector migrate` - Run database migrations
- `projector --verbose` - Enable verbose output

### API Endpoints

- `GET /api/actions` - List all actions
- `PUT /api/actions` - Create new action
- `GET /api/actions/:id` - Get action by ID
- `PUT /api/actions/:id` - Mark action as done
- `DELETE /api/actions/:id` - Delete action
- `GET /api/projects` - List all projects
- `PUT /api/projects` - Create new project
- `GET /api/projects/:id` - Get project by ID
- `DELETE /api/projects/:id` - Delete project
- `GET /health` - Health check

## Configuration

The application uses SQLite for data storage. The database file is created automatically in the current directory as `database.sqlite`.

## Development

### Prerequisites
- Go 1.24.5 or later
- SQLite3

### Building
```bash
go build -o projector
```

### Running Tests
```bash
go test ./...
```

## License

MIT License - see LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
