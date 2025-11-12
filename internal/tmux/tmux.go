// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package tmux

// IMPORTS {{{ 
import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/forms"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/mitchellh/go-homedir"
) // }}} 

const DefaultShell = "/bin/bash"

// GetTmuxSessionNames returns a slice of all active tmux session names.
// Returns an empty slice if tmux is not available or if there are no sessions.
func GetTmuxSessionNames() []string {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		// An error can occur if tmux isn't installed or if no server is running.
		// In either case, there are no sessions, so we return an empty slice
		// to provide a safe, non-nil value for callers.
		return []string{}
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return []string{} // No sessions found
	}

	lines := strings.Split(outputStr, "\n")

	// Pre-allocate the slice. len(lines) is a perfect capacity estimate
	// as tmux outputs one session name per line, preventing re-allocations.
	sessions := make([]string, 0, len(lines))
	for _, line := range lines {
		// This check handles cases of extraneous newlines in tmux output.
		if line != "" {
			sessions = append(sessions, line)
		}
	}

	return sessions
}

// GetSessionsExceptCurrent returns all tmux session names except the specified current session.
//
// This is useful for getting a list of sessions that can be switched to,
// excluding the session the user is currently in.
func GetSessionsExceptCurrent(current string) []string {
	sessions := GetTmuxSessionNames()
	if idx := slices.Index(sessions, current); idx >= 0 {
		sessions = slices.Delete(sessions, idx, idx+1)
	}
	return sessions
}

// HasTmuxSession checks if a tmux session with the given name exists.
func HasTmuxSession(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

// GetTmuxSessionSet returns a set (map[string]bool) of active session names
// for efficient membership testing when you need to check many sessions.
func GetTmuxSessionSet() map[string]bool {
	names := GetTmuxSessionNames()
	// PERF: Pre-allocate map with exact capacity to avoid rehashing
	sessions := make(map[string]bool, len(names))
	for _, name := range names {
		sessions[name] = true
	}

	return sessions
}

// GetCurrentTmuxSession returns the name of the current tmux session.
// Returns an empty string if not running inside tmux or if there's
// an error retrieving the session name.
func GetCurrentTmuxSession() string {
	if os.Getenv(constants.EnvTmux) == "" {
		return ""
	}

	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// SwitchToExistingSession switches to an existing tmux session by name.
// This function assumes the session already exists and will return an error if it doesn't.
// It handles both cases of running inside tmux (switch-client) and outside tmux (attach-session).
func SwitchToExistingSession(cfg *models.Config, name string) error {
	if !HasTmuxSession(name) {
		return fmt.Errorf("session '%s' does not exist", name)
	}

	target := getSessionTarget(cfg, name)

	if os.Getenv(constants.EnvTmux) == "" {
		return attachToSession(target, name)
	} else {
		return switchClientToSession(target, name)
	}
}

// IsTmuxServerRunning checks if a tmux server is currently running
func IsTmuxServerRunning() bool {
	cmd := exec.Command("tmux", "list-sessions")
	return cmd.Run() == nil
}

// CreateAndSwitchSession creates a new tmux session and switches to it.
// If the session already exists, it just switches to it.
func CreateAndSwitchSession(cfg *models.Config, session models.Session) error {
	if HasTmuxSession(session.Name) {
		return SwitchToExistingSession(cfg, session.Name)
	}

	if err := CreateSession(session); err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	return SwitchToExistingSession(cfg, session.Name)
}

// getSessionTarget returns the target string for tmux commands,
// incorporating the tmux_base configuration for window targeting.
func getSessionTarget(cfg *models.Config, name string) string {
	tmuxBase := cfg.TmuxBase
	if tmuxBase >= 0 {
		return fmt.Sprintf("%s:%d", name, tmuxBase)
	}

	return name
}

// attachToSession attaches to a session when not currently in tmux
func attachToSession(target, fallbackName string) error {
	// Check if server is running before attempting to attach
	if !IsTmuxServerRunning() {
		os.Exit(0) // Exit gracefully if no server
	}

	cmd := exec.Command("tmux", "attach-session", "-t", target)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		// If targeting a specific window failed, try just the session name
		if target != fallbackName {
			cmd := exec.Command("tmux", "attach-session", "-t", fallbackName)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			err := cmd.Run()
			if err == nil {
				os.Exit(0) // Exit successfully after attaching
			}
			// If attach failed and server is not running, exit gracefully
			if !IsTmuxServerRunning() {
				os.Exit(0)
			}
			return err
		}

		// If attach failed and server is not running, exit gracefully
		if !IsTmuxServerRunning() {
			os.Exit(0)
		}
		return fmt.Errorf("attaching to session: %w", err)
	}

	// Exit successfully after attaching to the session
	os.Exit(0)
	return nil // This line is never reached but required for compilation
}

// switchClientToSession switches to a session when already in tmux
func switchClientToSession(target, fallbackName string) error {
	if err := exec.Command("tmux", "switch-client", "-t", target).Run(); err != nil {
		// If targeting a specific window failed, try just the session name
		if target != fallbackName {
			return exec.Command("tmux", "switch-client", "-t", fallbackName).Run()
		}

		return fmt.Errorf("switching to session: %w", err)
	}

	return nil
}

// CreateSession creates a new tmux session using the provided session configuration.
// The session layout must contain at least one window definition.
//
// The first window is created with the new-session command, and subsequent
// windows are added using new-window. Each window can optionally specify a
// command to run upon creation.
func CreateSession(session models.Session) error {
	if len(session.Layout.Windows) == 0 {
		return fmt.Errorf("no windows defined in session layout")
	}

	// REFACTOR: Consider using a single tmux command with multiple operations for better performance
	w0 := session.Layout.Windows[0]
	args := buildWindowArgs(true, session.Name, w0.Name, session.Path, w0.Cmd)
	if err := exec.Command("tmux", args...).Run(); err != nil {
		return err
	}

	for _, w := range session.Layout.Windows[1:] {
		args := buildWindowArgs(false, session.Name, w.Name, session.Path, w.Cmd)
		if err := exec.Command("tmux", args...).Run(); err != nil {
			return err
		}
	}

	return nil
}

// buildWindowArgs constructs tmux command arguments for creating a window.
//
// For the first window it uses new-session, for subsequent windows it uses new-window.
// If cmd is provided, it wraps it with shell execution to keep the window open.
func buildWindowArgs(isFirst bool, sessionName, windowName, dir, cmd string) []string {
	var args []string
	if isFirst {
		args = []string{"new-session", "-ds", sessionName, "-n", windowName, "-c", dir}
	} else {
		args = []string{"new-window", "-t", sessionName, "-n", windowName, "-c", dir}
	}

	if cmd != "" {
		shell := os.Getenv(constants.EnvShell)
		if shell == "" {
			shell = DefaultShell
		}
		cmdStr := cmd + "; exec " + shell
		args = append(args, cmdStr)
	}

	return args
}