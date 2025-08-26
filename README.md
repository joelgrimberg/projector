# Projector

A CLI application for project and action management with a modern terminal UI.

## Features

- **Project Management**: Create, view, and manage projects
- **Action Tracking**: Add actions to projects with due dates, notes, and status
- **Status Management**: Track action progress (Not Started, In Progress, Done)
- **Tagging System**: Organize actions with custom tags
- **REST API**: Full HTTP API for integration with other tools
- **Interactive TUI**: Beautiful terminal-based user interface
- **Cross-Platform**: Works on macOS, Linux, and Windows
- **Persistent Storage**: SQLite database stored in `~/.local/share/projector/`

## Installation

### Via Homebrew (Recommended)

```
```

## Configuration

The application uses SQLite for data storage. The database file is automatically created in `~/.local/share/projector/projector.db` on all platforms.

### Database Location

- **Path**: `~/.local/share/projector/projector.db`
- **Permissions**: User read/write (0755)
- **Auto-creation**: Directory and database are created automatically on first run
- **Backup-friendly**: Standard backup tools include this location

### Custom Database Path

You can override the database path by setting the `PROJECTOR_DB_PATH` environment variable:

```bash
export PROJECTOR_DB_PATH="/custom/path/projector.db"
projector
```