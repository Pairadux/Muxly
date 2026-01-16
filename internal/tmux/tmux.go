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
// Returns an empty slice if tmux is not available or if there's an error.
func GetTmuxSessionNames() []string {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	// PERF: Pre-allocate sessions slice with estimated capacity based on typical session count
	var sessions []string
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
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
		args = append(args, "--", shell, "-lc", cmdStr)
	}

	return args
}

// KillSession terminates the specified tmux session.
//
// Returns an error if the session doesn't exist or if the kill operation fails.
func KillSession(target string) error {
	if err := exec.Command("tmux", "kill-session", "-t", target).Run(); err != nil {
		return fmt.Errorf("killing session: %w", err)
	}

	return nil
}

// KillServer terminates the entire tmux server and all sessions.
//
// This is a destructive operation that will close all tmux sessions.
// Returns an error if the kill operation fails.
func KillServer() error {
	if err := exec.Command("tmux", "kill-server").Run(); err != nil {
		return fmt.Errorf("killing server: %w", err)
	}

	return nil
}

// CreateAndSwitchToFallbackSession creates and switches to the configured fallback session.
// If no fallback session is configured, it uses "default" as the session name.
// The session is created in the user's home directory with the configured layout.
//
// Returns an error if session creation or switching fails.
func CreateAndSwitchToFallbackSession(cfg *models.Config) error {
	sessionName := cfg.FallbackSession.Name
	if sessionName == "" {
		sessionName = "Default"
	}

	if HasTmuxSession(sessionName) {
		return SwitchToExistingSession(cfg, sessionName)
	}

	sessionPath := cfg.FallbackSession.Path
	if sessionPath == "" {
		var err error
		sessionPath, err = homedir.Dir()
		if err != nil {
			return fmt.Errorf("failed to get homedir: %w", err)
		}
	}

	sessionLayout := cfg.FallbackSession.Layout
	if len(sessionLayout.Windows) == 0 {
		sessionLayout = cfg.SessionLayout
	}

	session := models.Session{
		Name:   sessionName,
		Path:   sessionPath,
		Layout: sessionLayout,
	}

	if err := CreateAndSwitchSession(cfg, session); err != nil {
		return fmt.Errorf("failed to create and switch to session '%s': %w", sessionName, err)
	}

	return nil
}

func CreateSessionFromForm(cfg models.Config) error {
	var (
		useFallback   bool
		confirmCreate bool
		sessionName   string
		path          string
		windowsStr    string
	)

	form := forms.CreateForm(&useFallback, &confirmCreate, &sessionName, &path, &windowsStr)
	if err := form.Run(); err != nil {
		return fmt.Errorf("form error: %w", err)
	}

	if !confirmCreate {
		return nil
	}

	if useFallback {
		return CreateAndSwitchToFallbackSession(&cfg)
	}

	if HasTmuxSession(sessionName) {
		return fmt.Errorf("session '%s' already exists", sessionName)
	}

	layout := parseWindows(windowsStr)
	if len(layout.Windows) == 0 {
		layout = cfg.SessionLayout
	}

	session := models.Session{
		Name:   sessionName,
		Path:   path,
		Layout: layout,
	}

	return CreateAndSwitchSession(&cfg, session)
}

// parseWindows parses a newline-delimited input string where each line is a name:cmd pair.
//
// It converts each name:cmd pair into Window structs for the session layout.
// If no colon is found in a line, the entire line is treated as the window name with no command.
// Returns a SessionLayout with parsed windows, or empty layout if input is empty.
func parseWindows(input string) models.SessionLayout {
	input = strings.TrimSpace(input)
	if input == "" {
		return models.SessionLayout{}
	}

	var windows []models.Window
	lines := strings.Split(input, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}

		var cmd string
		if len(parts) > 1 {
			cmd = strings.TrimSpace(parts[1])
		}

		windows = append(windows, models.Window{
			Name: name,
			Cmd:  cmd,
		})
	}

	return models.SessionLayout{Windows: windows}
}
