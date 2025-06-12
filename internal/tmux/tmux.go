// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package tmux

// IMPORTS {{{
import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Pairadux/tms/internal/models"

	"github.com/spf13/viper"
) // }}}

// GetTmuxSessionNames returns a slice of all active tmux session names.
// Returns an empty slice if tmux is not available or if there's an error.
func GetTmuxSessionNames() []string {
	if err := ValidateTmuxAvailable(); err != nil {
		return nil
	}

	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var sessions []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			sessions = append(sessions, line)
		}
	}

	return sessions
}

// HasTmuxSession checks if a tmux session with the given name exists.
func HasTmuxSession(name string) bool {
	if err := ValidateTmuxAvailable(); err != nil {
		return false
	}

	return exec.Command("tmux", "has-session", "-t", name).Run() == nil
}

// GetTmuxSessionSet returns a set (map[string]bool) of active session names
// for efficient membership testing when you need to check many sessions.
func GetTmuxSessionSet() map[string]bool {
	sessions := make(map[string]bool)
	names := GetTmuxSessionNames()
	for _, name := range names {
		sessions[name] = true
	}

	return sessions
}

// GetCurrentTmuxSession returns the name of the current tmux session.
// Returns an empty string if not running inside tmux or if there's
// an error retrieving the session name.
func GetCurrentTmuxSession() string {
	if os.Getenv("TMUX") == "" {
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
func SwitchToExistingSession(name string) error {
	if err := ValidateTmuxAvailable(); err != nil {
		return err
	}

	if !HasTmuxSession(name) {
		return fmt.Errorf("session '%s' does not exist", name)
	}

	target := getSessionTarget(name)

	if os.Getenv("TMUX") == "" {
		return attachToSession(target, name)
	} else {
		return switchClientToSession(target, name)
	}
}

// CreateAndSwitchSession creates a new tmux session and switches to it.
// If the session already exists, it just switches to it.
func CreateAndSwitchSession(name, cwd string) error {
	if err := ValidateTmuxAvailable(); err != nil {
		return err
	}

	if HasTmuxSession(name) {
		return SwitchToExistingSession(name)
	}

	var sessionLayout models.SessionLayout
	if err := viper.UnmarshalKey("session_layout", &sessionLayout); err != nil {
		return fmt.Errorf("failed to decode session_layout: %w", err)
	}

	if err := CreateSession(sessionLayout, name, cwd); err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	return SwitchToExistingSession(name)
}

// getSessionTarget returns the target string for tmux commands,
// incorporating the tmux_base configuration for window targeting.
func getSessionTarget(name string) string {
	tmuxBase := viper.GetInt("tmux_base")
	if tmuxBase >= 0 {
		return fmt.Sprintf("%s:%d", name, tmuxBase)
	}

	return name
}

// attachToSession attaches to a session when not currently in tmux
func attachToSession(target, fallbackName string) error {
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
			return cmd.Run()
		}

		return fmt.Errorf("attaching to session: %w", err)
	}

	return nil
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

// CreateSession creates a new tmux session with the specified name and working
// directory, using the provided session layout configuration. The session layout
// must contain at least one window definition.
//
// The first window is created with the new-session command, and subsequent
// windows are added using new-window. Each window can optionally specify a
// command to run upon creation.
func CreateSession(sessionLayout models.SessionLayout, session, dir string) error {
	if len(sessionLayout.Windows) == 0 {
		return fmt.Errorf("no windows defined in session layout")
	}

	w0 := sessionLayout.Windows[0]
	args := buildWindowArgs(true, session, w0.Name, dir, w0.Cmd)
	if err := exec.Command("tmux", args...).Run(); err != nil {
		return err
	}

	for _, w := range sessionLayout.Windows[1:] {
		args := buildWindowArgs(false, session, w.Name, dir, w.Cmd)
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
func buildWindowArgs(isFirst bool, session, windowName, dir, cmd string) []string {
	var args []string
	if isFirst {
		args = []string{"new-session", "-ds", session, "-n", windowName, "-c", dir}
	} else {
		args = []string{"new-window", "-t", session, "-n", windowName, "-c", dir}
	}

	if cmd != "" {
		shell := os.Getenv("SHELL")
		if shell == "" {
			shell = "/bin/bash"
		}
		cmdStr := cmd + "; exec " + shell
		args = append(args, "--", shell, "-lc", cmdStr)
	}

	return args
}

// ValidateTmuxAvailable checks if the tmux command is available in the system PATH.
//
// Returns an error if tmux is not found.
func ValidateTmuxAvailable() error {
	if _, err := exec.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}

	return nil
}
