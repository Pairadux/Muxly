<div align="center">
  <img src="./assets/muxly.png" alt="fzf - a command-line fuzzy finder">
  <a href="http://github.com/Pairadux/Muxly/releases"><img src="https://img.shields.io/github/v/tag/Pairadux/Muxly" alt="Version"></a>
  <a href="https://github.com/Pairadux/Muxly?tab=MIT-1-ov-file#readme"><img src="https://img.shields.io/github/license/Pairadux/Muxly" alt="License"></a>
  <a href="https://github.com/Pairadux/Muxly/graphs/contributors"><img src="https://img.shields.io/github/contributors/Pairadux/Muxly" alt="Contributors"></a>
  <a href="https://github.com/Pairadux/Muxly/stargazers"><img src="https://img.shields.io/github/stars/Pairadux/Muxly?style=flat" alt="Stars"></a>
</div>

---

A lightweight, highly customizable CLI for managing tmux sessions with ease!

## What is this?

Muxly is a highly configurable Tmux Session Manager based on ThePrimeagen's tmux-sessionizer script. It provides an intuitive interface for creating, managing, and switching between tmux sessions with pre-defined or on-the-fly layouts and intelligent directory scanning.

## Features

- **Interactive Session Selection**: Use `fzf` for fuzzy finding and selecting sessions
- **Intelligent Directory Scanning**: Automatically discover projects in configured directories  
- **Custom Session Layouts**: Define window configurations with specific commands for each session
- **Session Management**: Create, switch, and kill sessions with simple commands
- **Interactive TUI**: Create sessions with a beautiful terminal user interface
- **Configurable**: YAML-based configuration with sensible defaults
- **Tmux Integration**: Seamlessly integrates with existing tmux workflows

### Available Commands

- `muxly` - Interactive session selector with fzf
- `muxly create` - Interactive TUI for creating new sessions
- `muxly switch` - Switch between active tmux sessions
- `muxly kill` - Kill current session and switch to another
- `muxly config init` - Create initial configuration file
- `muxly config edit` - Edit configuration file

## Installation

### Prerequisites

- `tmux` - Terminal multiplexer
- `fzf` - Fuzzy finder for interactive selection

### Package Managers (Recommended)

| Platform | Command |
|----------|---------|
| **Arch Linux (AUR)** | `yay -S muxly` or `paru -S muxly` |
| **macOS (Homebrew)** | `brew install muxly` |

### Download Pre-built Binary

```bash
# Download and install the latest release (this one is for linux 64-bit systems)
wget -c https://github.com/Pairadux/muxly/releases/latest/download/muxly_Linux_x86_64.tar.gz -O - | tar xz
sudo chmod +x muxly
sudo mv muxly /usr/local/bin/
```

For other platforms, download the appropriate binary from the [releases page](https://github.com/Pairadux/muxly/releases).

### Build from Source

```bash
git clone https://github.com/Pairadux/muxly.git
cd muxly
go install
```

The binary will be installed as `muxly` in your `$GOPATH/bin` directory.

All examples in this README use `muxly` as the command name.

## Configuration

Configuration is stored in `$XDG_CONFIG_HOME/muxly/config.yaml` (typically `~/.config/muxly/config.yaml`).

### Initialize Configuration

```bash
muxly config init
```

### Configuration Options

```yaml
# Directories to scan for projects
scan_dirs:
  - path: ~/Dev
    depth: 1  # Optional: scanning depth (default: 1)
    alias: "" # Optional: display alias
  - path: ~/.dotfiles/dot_config

# Additional entry directories (always included)
entry_dirs:
  - ~/Documents
  - ~/Cloud

# Directory names to ignore when scanning
ignore_dirs:
  - ~/Dev/_practice
  - ~/Dev/_archive

# Default layout for new tmux sessions
session_layout:
  windows:
    - name: edit
      cmd: nvim    # Command to run in window
    - name: term
      cmd: ""      # Empty command opens shell

# Fallback session for when killing the final session
fallback_session:
  name: Default
  path: ~/
  layout:
    windows:
      - name: window
        cmd: ""

# Tmux configuration
tmux_base: 1                    # Base index for tmux windows (0 or 1)
default_depth: 1                # Default scanning depth
tmux_session_prefix: "[TMUX] "  # Prefix for active sessions

# Editor for config editing
editor: vi
```

### Configuration Details

#### Scan Directories

Scan directories are searched recursively for projects. Each directory can have:
- `path`: Directory path to scan (supports `~` expansion)
- `depth`: How deep to scan (optional, uses `default_depth` if not specified)
- `alias`: Display name in selector (optional)

#### Session Layouts  

Define default window configurations:
- `name`: Window name
- `cmd`: Command to run when window opens (optional)

Sessions can have custom layouts defined in the fallback session or applied globally.

## Usage Examples

### Basic Usage

```bash
# Launch interactive session selector
muxly

# Create a new session interactively
muxly create

# Switch between active sessions
muxly switch

# Kill current session and switch to another
muxly kill
```

### Configuration Management

```bash
# Create initial config
muxly config init

# Edit config file
muxly config edit
```

### Direct Session Creation

```bash
# Create/switch to session by name
muxly my-project
```

## Warning

This program is in a highly unstable state. The API and commands are subject to change before final release. The overall functionality of the program should be stable, unless otherwise stated though.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Pairadux/muxly&type=Date)](https://www.star-history.com/#Pairadux/muxly&Date)
