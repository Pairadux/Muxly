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

// GetTmuxSessions returns a map of all active tmux session names.
// The map values are always true; the map serves as a set for quick
// membership testing. Returns an empty map if tmux is not available
// or if there's an error listing sessions.
func GetTmuxSessions() map[string]bool {
	sessions := make(map[string]bool)
	if err := ValidateTmuxAvailable(); err != nil {
		return sessions
	}
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return sessions
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			sessions[line] = true
		}
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

// TmuxSwitchSession switches to or creates a tmux session with the given name
// and working directory. If the session doesn't exist, it creates it using the
// configured session layout. The function handles both cases of running inside
// an existing tmux session (uses switch-client) and outside tmux (uses attach-session).
//
// It attempts to target a specific window using the tmux_base configuration value,
// falling back to the session name if the specific window target fails.
func TmuxSwitchSession(name, cwd string) error {
	if err := ValidateTmuxAvailable(); err != nil {
		return err
	}
	var sessionLayout models.SessionLayout
	if err := viper.UnmarshalKey("session_layout", &sessionLayout); err != nil {
		return fmt.Errorf("failed to decode session_layout: %w", err)
	}
	sessionExists := exec.Command("tmux", "has-session", "-t", name).Run() == nil
	if !sessionExists {
		if err := CreateSession(sessionLayout, name, cwd); err != nil {
			return fmt.Errorf("creating session: %w", err)
		}
	}
	tmuxBase := viper.GetInt("tmux_base")
	target := fmt.Sprintf("%s:%d", name, tmuxBase)
	if os.Getenv("TMUX") == "" {
		cmd := exec.Command("tmux", "attach-session", "-t", target)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			if target != name {
				cmd := exec.Command("tmux", "attach-session", "-t", name)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				return cmd.Run()
			}
			return fmt.Errorf("attaching to session: %w", err)
		}
	} else {
		if err := exec.Command("tmux", "switch-client", "-t", target).Run(); err != nil {
			if target != name {
				return exec.Command("tmux", "switch-client", "-t", name).Run()
			}
			return fmt.Errorf("switching to session: %w", err)
		}
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
	args := []string{"new-session", "-ds", session, "-n", w0.Name, "-c", dir}
	if w0.Cmd != "" {
		args = append(args, strings.Fields(w0.Cmd)...)
	}
	if err := exec.Command("tmux", args...).Run(); err != nil {
		return err
	}
	for _, w := range sessionLayout.Windows[1:] {
		args = []string{"new-window", "-t", session, "-n", w.Name, "-c", dir}
		if w.Cmd != "" {
			args = append(args, w.Cmd)
		}
		if err := exec.Command("tmux", args...).Run(); err != nil {
			return err
		}
	}
	return nil
}

// ValidateTmuxAvailable checks if the tmux command is available in the system PATH.
// Returns an error if tmux is not found.
func ValidateTmuxAvailable() error {
	if _, err := exec.LookPath("tmux"); err != nil {
		return fmt.Errorf("tmux not found in PATH: %w", err)
	}
	return nil
}
