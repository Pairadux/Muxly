package cmd

import (
	"errors"
	"fmt"
	"slices"

	"github.com/Pairadux/muxly/internal/forms"
	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/tmux"

	"github.com/spf13/cobra"
)

var killServer bool

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill [SESSION]",
	Short: "Kill the current tmux session and switch to another",
	Long: `Kill the current tmux session and switch to another.

If SESSION is provided, the client switches to it and the current session is killed.
SESSION must be the exact name of another session.
Otherwise, a picker list of active sessions is displayed to choose a replacement.
If no other sessions exist, you can create a new session from the default template.
Declining kills the tmux server, because killing the last session ends it anyway.

Outside tmux there is no current session, so only --kill-server does anything.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !tmux.IsTmuxServerRunning() {
			fmt.Println("No tmux server running. No changes made.")
			return nil
		}

		if killServer {
			return killTmuxServer()
		}

		currentSession := tmux.GetCurrentTmuxSession()
		if currentSession == "" {
			return errors.New("not inside a tmux session, so there is no session to kill\nRun 'muxly kill -k' to kill the tmux server")
		}

		otherSessions := tmux.GetSessionsExceptCurrent(currentSession)

		// Naming the current session is the same as naming none: it can't replace itself.
		var choiceStr string
		if len(args) == 1 && args[0] != currentSession {
			choiceStr = args[0]
			if !slices.Contains(otherSessions, choiceStr) {
				return fmt.Errorf("no other session is named %q\nRun 'muxly kill' to pick one", choiceStr)
			}
		}

		if choiceStr == "" {
			if len(otherSessions) == 0 {
				return killLastSession(currentSession)
			}

			var err error
			choiceStr, err = fzf.SelectWithFzf(otherSessions)
			if err != nil {
				if errors.Is(err, fzf.ErrCancelled) {
					return nil
				}
				return fmt.Errorf("selecting with fzf failed: %w", err)
			}

			if choiceStr == "" {
				return nil
			}
		}

		if err := tmux.SwitchToExistingSession(&state.Cfg, choiceStr); err != nil {
			if errors.Is(err, tmux.ErrGracefulExit) {
				return nil
			}
			return fmt.Errorf("failed to switch session: %w", err)
		}

		if err := tmux.KillSession(currentSession); err != nil {
			return fmt.Errorf("failed to kill session: %w", err)
		}

		return nil
	},
}

// killLastSession handles killing the only session left, which would also end the
// server. Unless the config says to just do that, it offers a fresh session instead.
func killLastSession(currentSession string) error {
	if state.Cfg.Settings.AlwaysKillOnLastSession {
		return killTmuxServer()
	}

	var createFromTemplate bool
	form := forms.ConfirmationForm("Create session from default template?", "Declining will kill the tmux server.", &createFromTemplate)
	if err := form.Run(); err != nil {
		if errors.Is(err, forms.ErrCancelled) {
			return nil
		}
		return fmt.Errorf("failed to run confirmation form: %w", err)
	}

	if !createFromTemplate {
		return killTmuxServer()
	}

	if err := tmux.ReplaceLastSession(&state.Cfg, currentSession); err != nil {
		return fmt.Errorf("failed to replace session: %w", err)
	}
	return nil
}

func killTmuxServer() error {
	if err := tmux.KillServer(); err != nil {
		return fmt.Errorf("failed to kill tmux server: %w", err)
	}
	fmt.Println("Tmux server killed.")
	return nil
}

func init() {
	rootCmd.AddCommand(killCmd)
	killCmd.Flags().BoolVarP(&killServer, "kill-server", "k", false, "Kill tmux server (rather than current session)")
}
