# Tmux-Sessionizer (tms)

A lightweight CLI for handling TMUX sessions with ease!

## What is this?

Tmux Sessionizer is a highly configurable Tmux Session Manager based on ThePrimeagen's tmux-sessionizer script. It provides an intuitive interface for creating, managing, and switching between tmux sessions with pre-defined layouts and intelligent directory scanning.

## Features

- **Interactive Session Selection**: Use `fzf` for fuzzy finding and selecting sessions
- **Intelligent Directory Scanning**: Automatically discover projects in configured directories  
- **Custom Session Layouts**: Define window configurations with specific commands for each session
- **Session Management**: Create, switch, and kill sessions with simple commands
- **Interactive TUI**: Create sessions with a beautiful terminal user interface
- **Configurable**: YAML-based configuration with sensible defaults
- **Tmux Integration**: Seamlessly integrates with existing tmux workflows

### Available Commands

- `tms` - Interactive session selector with fzf
- `tms create` - Interactive TUI for creating new sessions  
- `tms switch` - Switch between active tmux sessions
- `tms kill` - Kill current session and switch to another
- `tms config init` - Create initial configuration file
- `tms config edit` - Edit configuration file

## Installation

### Prerequisites

- `tmux` - Terminal multiplexer
- `fzf` - Fuzzy finder for interactive selection

### Download Pre-built Binary (Recommended)

```bash
# Download and install the latest release
wget -c https://github.com/Pairadux/Tmux-Sessionizer/releases/latest/download/Tmux-Sessionizer_Linux_x86_64.tar.gz -O - | tar xz
sudo chmod +x Tmux-Sessionizer
sudo mv Tmux-Sessionizer /usr/local/bin/
```

For other platforms, download the appropriate binary from the [releases page](https://github.com/Pairadux/Tmux-Sessionizer/releases).

### Package Managers

AUR and Homebrew packages are planned for the 1.0.0 stable release.

### Build from Source

```bash
git clone https://github.com/Pairadux/Tmux-Sessionizer.git
cd Tmux-Sessionizer
go install
```

The binary will be installed as `Tmux-Sessionizer` in your `$GOPATH/bin` directory.

### Recommended Alias

For convenience, it's recommended to create an alias for the binary:

```bash
# Add to your shell configuration (.bashrc, .zshrc, etc.)
alias tms='Tmux-Sessionizer'
```

All examples in this README use `tms` for brevity, but you can substitute `Tmux-Sessionizer` if you prefer not to use the alias.

## Configuration

Configuration is stored in `$XDG_CONFIG_HOME/tms/config.yaml` (typically `~/.config/tms/config.yaml`).

### Initialize Configuration

```bash
tms config init
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
tms

# Create a new session interactively  
tms create

# Switch between active sessions
tms switch

# Kill current session and switch to another
tms kill
```

### Configuration Management

```bash
# Create initial config
tms config init

# Edit config file
tms config edit
```

### Direct Session Creation

```bash
# Create/switch to session by name
tms my-project
```

## Warning

This program is in a highly unstable state. The API and commands are subject to change before final release. The overall functionality of the program should be stable, unless otherwise stated though.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Pairadux/Tmux-Sessionizer&type=Date)](https://www.star-history.com/#Pairadux/Tmux-Sessionizer&Date)
