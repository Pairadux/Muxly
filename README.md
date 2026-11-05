<div align="center">
  <img src="./assets/muxly.png" alt="fzf - a command-line fuzzy finder">
  <a href="http://github.com/Pairadux/muxly/releases"><img src="https://img.shields.io/github/v/tag/Pairadux/muxly" alt="Version"></a>
  <a href="https://github.com/Pairadux/muxly/blob/main/LICENSE"><img src="https://img.shields.io/github/license/Pairadux/muxly" alt="License"></a>
  <a href="https://github.com/Pairadux/muxly/graphs/contributors"><img src="https://img.shields.io/github/contributors/Pairadux/muxly" alt="Contributors"></a>
  <a href="https://github.com/Pairadux/muxly/actions/workflows/test.yml"><img src="https://img.shields.io/github/actions/workflow/status/Pairadux/muxly/test.yml?branch=main&label=tests" alt="Tests"></a>
  <a href="https://github.com/Pairadux/muxly/stargazers"><img src="https://img.shields.io/github/stars/Pairadux/muxly?style=flat" alt="Stars"></a>
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
- `muxly config add` - Add directories to configuration (entry or scan)
- `muxly config remove` - Remove directories from configuration
- `muxly config init` - Create initial configuration file (use `--force` to replace an existing one)
- `muxly config edit` - Edit configuration file
- `muxly config list` - List the directories in the configuration
- `muxly doctor` - Validate environment and configuration
- `muxly completion <shell>` - Generate shell completion scripts (hidden command)

## Installation

### Prerequisites

- `tmux` 3.2 or newer - Terminal multiplexer
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
- Add your project directories with `muxly config add scan ~/Dev` or edit the config manually
- Customize your `session_layout` with windows and commands you use frequently
- Try `muxly create` for an interactive TUI to create sessions on the fly

## Shell Completion

Muxly supports shell completion for command and flag suggestions. To enable it, run the appropriate command for your shell:

<details>
<summary><b>Bash</b></summary>

```bash
# Add to ~/.bashrc
echo 'eval "$(muxly completion bash)"' >> ~/.bashrc

# Or generate a completion file
muxly completion bash > /etc/bash_completion.d/muxly
```

</details>

<details>
<summary><b>Zsh</b></summary>

```bash
# Add to ~/.zshrc (make sure compinit is loaded)
echo 'eval "$(muxly completion zsh)"' >> ~/.zshrc

# Or add to your fpath
muxly completion zsh > "${fpath[1]}/_muxly"
```

</details>

<details>
<summary><b>Fish</b></summary>

```bash
# Add to fish completions directory
muxly completion fish > ~/.config/fish/completions/muxly.fish
```

</details>

<details>
<summary><b>PowerShell</b></summary>

```powershell
# Add to your PowerShell profile
muxly completion powershell >> $PROFILE
```

</details>

After adding the completion script, restart your shell or source your config file.

## Directory Naming Notes

The picker shows each directory's real name. Directories with the same name add parent folders until they differ, as with `Dev/src` and `Work/src`, and a scan directory's `alias` always comes first, as in `proj/unique`.

An `entry_dirs` path that doesn't exist shows as `notes [missing]`, and picking it explains what's wrong rather than opening anything.

A running session replaces its directory in the picker, however the session got its name. Sessions belong to a directory by path, so a session called `foo` in another folder doesn't hide your `foo` directory. If a session's name doesn't resemble its directory, the picker adds the directory: `[TMUX] api (Dev/backend)`. A session you started by hand in a directory counts too.

Muxly gives each new session a short, safe name based on that label, so it can always find the session again:

- **Lowercase, with only letters, digits, `-` and `_`**. `MyProject` becomes `myproject`, and `my project` becomes `my-project`
- **Dots**: Replaced with underscores. `my.project` becomes `my_project`
- **Anything else**: Replaced with dashes. `project:v2` becomes `project-v2`
- **Symbols at either end**: Removed. `.config` becomes `config`, and `$dollar` becomes `dollar`
- **Folders and aliases**: Joined with dashes. `Dev/src` becomes `dev-src`, and `proj/unique` becomes `proj-unique`
- **Name already taken**: A session elsewhere with the same name means this one adds its parent folder, then a number: `dev-foo`, then `foo-2`
- **Names you type**: `muxly foo.bar` opens a session called `foo_bar`, and finds it again next time. Names must match exactly, so `muxly foo` won't open a running `foobar`

Accents and other alphabets are kept. If nothing is left, as with `🚀`, the session is called `session`.

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
- `MUXLY_TMUX_SESSION_PREFIX` - Prefix for active sessions in selector
- `MUXLY_ALWAYS_KILL_ON_LAST_SESSION` - Skip fallback prompt (true/false)

### Configuration File

#### Default Configuration

Running `muxly config init` creates this minimal config and prints where it wrote it. It won't replace an existing file unless you pass `--force`:

```yaml
# Additional entry directories (included directly, not scanned)
entry_dirs:
  - path: ~

# Session templates (one must have default: true)
templates:
  - name: default
    label: Editor + Terminal
    default: true
    windows:
      - name: editor
      - name: term
  - name: minimal
    label: Single Window
    windows:
      - name: main
  - name: quick
    label: Quick Session
    path: ~/
    windows:
      - name: main

# General settings
settings:
  # Editor for 'muxly config edit' (falls back to $EDITOR, then 'vi')
  editor: vi
  # Default scanning depth for directories
  default_depth: 1
  # Prefix for active tmux sessions in the selector
  tmux_session_prefix: "[TMUX] "
  # Always kill tmux server on last session (skips fallback session prompt)
  always_kill_on_last_session: false
```

This works immediately - no customization needed! But you'll probably want to add your project directories...

**Tip:** Use `muxly config add scan ~/your-projects` to add directories, or `muxly config edit` to edit the config file manually.

#### Advanced Configuration Example

<details>
<summary><b>Example with custom directories, windows, and commands (click to expand)</b></summary>

```yaml
# Directories to scan for projects
# Each directory is scanned recursively up to the specified depth
scan_dirs:
  - path: ~/projects
    depth: 2           # Override default_depth for this directory
    alias: proj        # Display as "proj/project-name" in selector
  - path: ~/work       # Alias is optional
    depth: 1
  - path: ~/dotfiles   # Uses default_depth if not specified

# Additional entry directories (included directly, not scanned)
entry_dirs:
  - ~/Documents
  - ~/notes

# Directories to exclude from scanning (additive)
# .git and node_modules are always ignored — no need to list them
# Bare names match any directory with that name at any depth
# Paths match that resolved directory and everything inside it
ignore_dirs:
  - .vscode               # bare name — skips all .vscode dirs
  - .idea                 # bare name — skips all .idea dirs
  - target                # bare name — skips all target dirs (e.g. Rust builds)
  - ~/projects/archived   # path — skips this directory and everything inside it

# Session templates (one must have default: true)
templates:
  - name: dev
    label: Dev Workflow
    default: true
    windows:
      - name: edit
        cmd: nvim        # Opens nvim in first window
      - name: term
        cmd: ""          # Empty command opens default shell
      - name: git
        cmd: lazygit     # Opens lazygit in third window
  - name: minimal
    label: Single Window
    windows:
      - name: main

# General settings
settings:
  # Editor for 'muxly config edit' (falls back to $EDITOR, then 'vi')
  editor: nvim
  # Default scanning depth for scan_dirs (can be overridden per directory)
  default_depth: 1
  # Prefix for active tmux sessions in the selector
  tmux_session_prefix: "[TMUX] "
  # Always kill tmux server on last session (skips fallback session prompt)
  always_kill_on_last_session: false
```

**Note:** You can also manage these directories using commands instead of manual editing:
- `muxly config add scan ~/Dev --depth 2 --alias dev` - Add to scan_dirs
- `muxly config add entry ~/Documents` - Add to entry_dirs
- `muxly config remove scan ~/Dev` - Remove from scan_dirs
- `muxly config remove entry ~/Documents` - Remove from entry_dirs

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

When you create a session for `~/my-project`, Muxly will use this layout instead of the default template's windows. This is perfect for projects with unique workflows or specific commands.

**Notes:**
- The `.muxly` file only needs a `windows` array - all other settings come from your global config
- If no `.muxly` file exists, the default template's windows are used
- `.muxly` files are not scanned/discovered automatically - they only apply when you select that specific directory
- When removing an entry directory with `muxly config remove entry`, you'll be prompted about deleting its `.muxly` file (use `--keep` or `--delete` flags for non-interactive use)

#### Ignore Rules

The `ignore_dirs` list supports two matching styles, determined automatically by the entry format:

| Entry format | Example | Behavior |
|---|---|---|
| **Bare name** (no `/` or `~`) | `target`, `.vscode` | Matches any directory with that exact name at any depth during scanning. The directory and its contents are skipped entirely. |
| **Path** (contains `/` or `~`) | `~/projects/archived` | Resolved to an absolute path. That directory and everything inside it are hidden, including anything in `entry_dirs`. |

**Built-in filters:** `.git` and `node_modules` are always ignored and cannot be overridden. Any entries you add to `ignore_dirs` are **additive** on top of these built-in filters.

```yaml
ignore_dirs:
  - target                # bare name — skips all target dirs (e.g. Rust builds)
  - .vscode               # bare name — skips all .vscode dirs
  - ~/projects/archived   # path — skips this directory and everything inside it
```

### Configuration Options Reference

| Option | Type | Required | Description |
|--------|------|----------|-------------|
| `scan_dirs` | array | yes* | Directories to scan for projects |
| `scan_dirs[].path` | string | yes | Directory path to scan (supports `~` and environment variables) |
| `scan_dirs[].depth` | int | no | Scan depth for this directory, at least 1 (overrides `settings.default_depth`) |
| `scan_dirs[].alias` | string | no | Prefix for directories from here, in the picker and in session names (e.g., "dev" shows as "dev/project-name" and names the session "dev-project-name") |
| `scan_dirs[].template` | string | no | Template name to use for sessions created from this scan directory |
| `entry_dirs` | array | yes* | Directories always included without scanning |
| `entry_dirs[].path` | string | yes | Directory path (supports `~` and environment variables) |
| `entry_dirs[].template` | string | no | Template name to use for sessions created from this directory |
| `ignore_dirs` | array | no | Additional directories to exclude from scanning, additive to built-in `.git`/`node_modules` filters (see [Ignore Rules](#ignore-rules)) |
| `templates` | array | yes | Session templates (exactly one must have `default: true`) |
| `templates[].name` | string | yes | Short identifier used as the tmux session name |
| `templates[].label` | string | no | Human-readable display name shown in `muxly create` TUI |
| `templates[].default` | bool | no | Mark exactly one template as the default (required on one) |
| `templates[].path` | string | no | Fixed working directory (uses fzf picker if omitted) |
| `templates[].windows` | array | yes | List of windows to create (must have at least one) |
| `templates[].windows[].name` | string | yes | Window name |
| `templates[].windows[].cmd` | string | no | Command to run in window (empty string opens default shell) |
| `settings` | object | no | General application settings |
| `settings.editor` | string | no | Editor for config editing, falls back to `$EDITOR` (default: `"vi"`) |
| `settings.default_depth` | int | no | Default scanning depth for `scan_dirs` (default: `1`) |
| `settings.tmux_session_prefix` | string | no | Prefix for active sessions in selector (default: `"[TMUX] "`) |
| `settings.always_kill_on_last_session` | bool | no | Skip fallback prompt and kill server on last session (default: `false`) |

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

# List the configured directories
muxly config list

# Add directories to scan_dirs
muxly config add scan ~/Dev --depth 2 --alias dev
muxly config add scan ~/projects

# Add directories to entry_dirs
muxly config add entry ~/Documents
muxly config add entry .

# Remove directories
muxly config remove scan ~/projects
muxly config remove entry ~/Documents
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
