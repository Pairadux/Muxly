# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

### Build and Run
- `go install` - Build the binary
- `go run main.go` - Run directly with go 
- `Tmux-Sessionizer` - Run the built binary

### Development Commands
- `go mod tidy` - Clean up dependencies
- `go fmt ./...` - Format all Go files
- `go vet ./...` - Static analysis
- `go test ./...` - Run tests (if any exist)

### Common Usage
- `Tmux-Sessionizer` - Interactive session selector with fzf
- `Tmux-Sessionizer config init` - Create initial config file
- `Tmux-Sessionizer config edit` - Edit config file
- `Tmux-Sessionizer create` - Interactive TUI for creating sessions
- `Tmux-Sessionizer switch` - Switch between active sessions  
- `Tmux-Sessionizer kill` - Kill current session and switch to another

## Architecture

This is a Go CLI application using the Cobra framework for command structure. The project follows a standard Go project layout:

### Core Components

**Main Entry Point**
- `main.go` - Simple entry point that calls `cmd.Execute()`
- `cmd/root.go` - Root Cobra command with main application logic

**Command Structure**
- `cmd/` - All CLI commands (create, kill, switch, config, etc.)
- Uses Cobra for command-line parsing and subcommands
- Config management through Viper with YAML files

**Internal Packages**
- `internal/models/` - Configuration structs and data models
- `internal/tmux/` - Tmux session management and commands
- `internal/fzf/` - Integration with fzf for interactive selection
- `internal/utility/` - Path resolution and directory traversal helpers
- `internal/forms/` - TUI forms using Charm's Huh library

### Key Architecture Patterns

**Configuration System**
- YAML-based config at `$XDG_CONFIG_HOME/tms/config.yaml`
- Supports `scan_dirs` (with depth control) and `entry_dirs`
- Session layouts with windows and commands
- Fallback session configuration

**Directory Discovery**
- Scans configured directories at specified depths using `fastwalk`
- Filters out ignored directories and current tmux session
- Combines with existing tmux sessions for selection

**Session Management**
- Creates tmux sessions with custom layouts
- Handles session switching and cleanup
- Integrates with existing tmux sessions

**External Dependencies**
- Requires `tmux` and `fzf` to be installed
- Validates external tools at startup (except for config commands)

### Key Files to Understand

- `cmd/root.go:67-132` - Main application logic and directory entry building
- `internal/models/models.go` - Configuration structure and types
- `internal/tmux/tmux.go` - Tmux session operations
- `cmd/root.go:194-272` - Directory entry building algorithm
