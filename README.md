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

### Quick Start

```bash
# Create default configuration
muxly config init

# Edit configuration
muxly config edit
```

### Configuration Precedence

Muxly uses the following precedence order (highest to lowest):
1. **CLI Flags** (e.g., `--depth 2`)
2. **Environment Variables** (e.g., `MUXLY_DEFAULT_DEPTH=3`)
3. **Config File** (`~/.config/muxly/config.yaml`)
4. **Built-in Defaults**

### Environment Variables

Override config values temporarily using environment variables:

```bash
# Override editor for a single command
MUXLY_EDITOR=emacs muxly config edit

# Override scan depth
MUXLY_DEFAULT_DEPTH=3 muxly

# Override tmux session prefix
MUXLY_TMUX_SESSION_PREFIX="[WORK] " muxly

# Standard $EDITOR works too (fallback if MUXLY_EDITOR not set)
EDITOR=nano muxly config edit
```

**Supported Environment Variables:**
- `MUXLY_EDITOR` or `EDITOR` - Editor for config editing
- `MUXLY_DEFAULT_DEPTH` - Default scanning depth
- `MUXLY_TMUX_BASE` - Tmux window base index
- `MUXLY_TMUX_SESSION_PREFIX` - Prefix for active sessions in selector
- `MUXLY_ALWAYS_KILL_ON_LAST_SESSION` - Skip fallback prompt (true/false)

### Configuration File

<details>
<summary><b>Complete Configuration Example (click to expand)</b></summary>

```yaml
# Directories to scan for projects
# Each directory is scanned recursively up to the specified depth
scan_dirs:
  - path: ~/Dev
    depth: 2           # Override default_depth for this directory
    alias: dev         # Display as "dev/project-name" in selector
  - path: ~/.config
    depth: 1
    alias: config
  - path: ~/projects   # Uses default_depth if not specified

# Additional entry directories (included directly, not scanned)
entry_dirs:
  - ~/Documents
  - ~/Cloud

# Directory paths to exclude from scanning
ignore_dirs:
  - ~/Dev/_practice
  - ~/Dev/_archive
  - ~/Dev/tmp

# Session to create when killing the last tmux session
fallback_session:
  name: Default
  path: ~/
  layout:
    windows:
      - name: shell
        cmd: ""

# Default layout for new tmux sessions
session_layout:
  windows:
    - name: edit
      cmd: nvim        # Opens nvim in first window
    - name: term
      cmd: ""          # Empty command opens default shell
    - name: git
      cmd: lazygit     # Opens lazygit in third window

# Base index for tmux windows (should match your tmux.conf)
# See: https://www.man7.org/linux/man-pages/man1/tmux.1.html#OPTIONS (base-index)
tmux_base: 1

# Default scanning depth for scan_dirs (can be overridden per directory)
default_depth: 1

# Prefix for active tmux sessions in the selector
tmux_session_prefix: "[TMUX] "

# Editor for 'muxly config edit' (falls back to $EDITOR, then 'vi')
editor: nvim

# Always kill tmux server on last session (skips fallback session prompt)
always_kill_on_last_session: false
```

</details>

### Configuration Options Reference

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `scan_dirs` | array | yes* | Directories to scan for projects |
| `scan_dirs[].path` | string | yes | Directory path to scan (supports `~` and environment variables) |
| `scan_dirs[].depth` | int | no | Scan depth for this directory (overrides `default_depth`) |
| `scan_dirs[].alias` | string | no | Display prefix in selector (e.g., "dev" shows as "dev/project-name") |
| `entry_dirs` | array | yes* | Directories always included without scanning |
| `ignore_dirs` | array | no | Paths to exclude from scanning |
| `fallback_session` | object | no | Session created when killing the last tmux session |
| `fallback_session.name` | string | no | Session name (default: `"Default"`) |
| `fallback_session.path` | string | no | Working directory (default: `"~/"`) |
| `fallback_session.layout` | object | no | Window layout (defaults to `session_layout`) |
| `session_layout` | object | yes | Default window layout for new sessions |
| `session_layout.windows` | array | yes | List of windows to create (must have at least one) |
| `session_layout.windows[].name` | string | yes | Window name |
| `session_layout.windows[].cmd` | string | no | Command to run in window (empty string opens default shell) |
| `tmux_base` | int | no | Tmux window [base index](https://www.man7.org/linux/man-pages/man1/tmux.1.html#OPTIONS) - 0 or 1, should match your tmux.conf (default: `1`) |
| `default_depth` | int | no | Default scanning depth for `scan_dirs` (default: `1`) |
| `tmux_session_prefix` | string | no | Prefix for active sessions in selector (default: `"[TMUX] "`) |
| `editor` | string | no | Editor for config editing, falls back to `$EDITOR` (default: `"vi"`) |
| `always_kill_on_last_session` | bool | no | Skip fallback prompt and kill server on last session (default: `false`) |

\* At least one of `scan_dirs` or `entry_dirs` must be configured.

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
