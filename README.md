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

## Why Muxly?

- **Simple by default**: Works immediately with minimal setup - just your home directory
- **Flexible when needed**: YAML config for advanced workflows, environment variable overrides for quick tweaks
- **Fuzzy finder first**: Built around `fzf` for lightning-fast session selection
- **Universal design**: No assumptions about your editors, tools, or workflow
- **Intelligent discovery**: Finds your projects automatically with configurable scanning
- **Built-in validation**: Config errors are caught immediately with helpful feedback

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
- `muxly add` - Add directories to configuration (entry or scan)
- `muxly remove` - Remove directories from configuration
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

## Getting Started

Get up and running with Muxly in 3 simple steps:

```bash
# 1. Create your config file (creates minimal defaults)
muxly config init

# 2. Customize your config (optional - works with defaults!)
muxly config edit

# 3. Launch Muxly and select a directory
muxly
```

That's it! Muxly will show you your home directory by default. Use the arrow keys or fuzzy search to select a directory, then press Enter to create or switch to that session.

**Next Steps:**
- Add your project directories with `muxly add scan ~/Dev` or edit the config manually
- Customize your `session_layout` with windows and commands you use frequently
- Try `muxly create` for an interactive TUI to create sessions on the fly

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

#### Default Configuration

Running `muxly config init` creates this minimal config:

```yaml
# Additional entry directories (included directly, not scanned)
entry_dirs:
  - ~

# Fallback session for when killing the final session
fallback_session:
  name: Default
  path: ~/
  layout:
    windows:
      - name: main
        cmd: ""

# Base index for tmux windows (0 or 1)
# See: https://www.man7.org/linux/man-pages/man1/tmux.1.html#OPTIONS (base-index)
tmux_base: 1

# Default scanning depth for directories
default_depth: 1

# Default layout for new tmux sessions
session_layout:
  windows:
    - name: main
      cmd: ""

# Prefix for active tmux sessions in the selector
tmux_session_prefix: "[TMUX] "

# Editor for 'muxly config edit' (falls back to $EDITOR, then 'vi')
editor: vi

# Always kill tmux server on last session (skips fallback session prompt)
always_kill_on_last_session: false
```

This works immediately - no customization needed! But you'll probably want to add your project directories...

**Tip:** Use `muxly add scan ~/your-projects` to add directories, or `muxly config edit` to edit the config file manually.

#### Advanced Configuration Example

<details>
<summary><b>Example with custom directories, windows, and commands (click to expand)</b></summary>

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
  - ~/special-project

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
      - name: main
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

**Note:** You can also manage these directories using commands instead of manual editing:
- `muxly add scan ~/Dev --depth 2 --alias dev` - Add to scan_dirs
- `muxly add entry ~/Documents` - Add to entry_dirs
- `muxly remove scan ~/Dev` - Remove from scan_dirs
- `muxly remove entry ~/Documents` - Remove from entry_dirs

See [Configuration Management](#configuration-management) examples below.

</details>

#### Directory-Specific Layouts

You can override the default session layout for specific projects by creating a `.muxly` file in the project directory:

**Example: `~/my-project/.muxly`**

```yaml
windows:
  - name: editor
    cmd: nvim src/
  - name: server
    cmd: npm run dev
  - name: tests
    cmd: npm run test:watch
```

When you create a session for `~/my-project`, Muxly will use this layout instead of the global `session_layout` from your config file. This is perfect for projects with unique workflows or specific commands.

**Notes:**
- The `.muxly` file only needs a `windows` array - all other settings come from your global config
- If no `.muxly` file exists, the global `session_layout` is used
- `.muxly` files are not scanned/discovered automatically - they only apply when you select that specific directory
- When removing an entry directory with `muxly remove entry`, you'll be prompted about deleting its `.muxly` file (use `--keep` or `--delete` flags for non-interactive use)

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

# Add directories to scan_dirs
muxly add scan ~/Dev --depth 2 --alias dev
muxly add scan ~/projects

# Add directories to entry_dirs
muxly add entry ~/Documents
muxly add entry .

# Remove directories
muxly remove scan ~/projects
muxly remove entry ~/Documents
```

### Direct Session Creation

```bash
# Create/switch to session by name
muxly my-project
```

## Project Status

Muxly is currently in **active development**. While the core functionality is stable and usable for daily workflows, the API and commands may evolve before the 1.0 release based on user feedback and feature requests.

**Current State:**
- ✅ Core features are stable and tested
- ✅ Safe for daily use
- ⚠️ Config structure may receive additions or improvements
- ⚠️ Command flags/options may change

We follow semantic versioning and will clearly communicate any breaking changes in release notes.

## Community & Contributing

Found a bug? Have a feature request? Want to contribute?

- **Issues**: [Report bugs or request features](https://github.com/Pairadux/muxly/issues)
- **Discussions**: [Ask questions or share ideas](https://github.com/Pairadux/muxly/discussions)
- **Contributing**: Pull requests are welcome! Please open an issue first to discuss major changes

We'd love to hear how you're using Muxly and what would make it better for your workflow.

## Star History

[![Star History Chart](https://api.star-history.com/svg?repos=Pairadux/muxly&type=Date)](https://www.star-history.com/#Pairadux/muxly&Date)
