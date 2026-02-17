package cmd

import (
	"errors"
	"fmt"

	"github.com/Pairadux/muxly/internal/forms"
	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/tmux"

	"github.com/spf13/cobra"
)

var killServer bool

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill [SESSION]",
	Short: "Kill a tmux session and switch to another",
	Long: `Kill a tmux session and switch to another.

If SESSION is provided, the current session is killed and the client switches to SESSION.
Otherwise, a picker list of active sessions is displayed to choose a replacement.
If no other sessions exist, a new session is created from the primary template or the tmux server is killed.`,
	Args:    cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		currentSession := tmux.GetCurrentTmuxSession()

		if currentSession == "" || killServer {
			var killServer bool
			form := forms.ConfirmationForm("Kill tmux server?", "This will terminate all tmux sessions.", &killServer)

			if err := form.Run(); err != nil {
				return fmt.Errorf("failed to run confirmation form: %w", err)
			}

			if !killServer {
				fmt.Println("Aborting. No changes made.")
				return nil
			}

			if err := tmux.KillServer(); err != nil {
				return fmt.Errorf("failed to kill tmux server: %w", err)
			}
			fmt.Println("tmux server killed.")
			return nil
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			sessions := tmux.GetSessionsExceptCurrent(currentSession)

			// IDEA: add config option to allow users to create new session rather than dropping back to existing one on kill
			// might even just make this the default behavior...
			if len(sessions) == 0 {
				if cfg.Settings.AlwaysKillOnLastSession {
					if err := tmux.KillServer(); err != nil {
						return fmt.Errorf("failed to kill tmux server: %w", err)
					}
					fmt.Println("tmux server killed.")
					return nil
				}

				var createFromTemplate bool
				form := forms.ConfirmationForm("Create session from primary template?", "Declining will kill the tmux server.", &createFromTemplate)

				if err := form.Run(); err != nil {
					return fmt.Errorf("failed to run confirmation form: %w", err)
				}

				if createFromTemplate {
					if err := tmux.CreateSessionFromPrimaryTemplate(&cfg); err != nil {
						if errors.Is(err, tmux.ErrGracefulExit) {
							return nil
						}
						return fmt.Errorf("failed to create session from primary template: %w", err)
					}
					if err := tmux.KillSession(currentSession); err != nil {
						return fmt.Errorf("failed to kill session: %w", err)
					}
				} else {
					if err := tmux.KillServer(); err != nil {
						return fmt.Errorf("failed to kill tmux server: %w", err)
					}
					fmt.Println("tmux server killed.")
				}

				return nil
			}

			var err error
			choiceStr, err = fzf.SelectWithFzf(sessions)
			if err != nil {
				if err.Error() == "user cancelled" {
					return nil
				}
				return fmt.Errorf("selecting with fzf failed: %w", err)
			}

			if choiceStr == "" {
				return nil
			}
		}
		sessionName := choiceStr
		if err := tmux.SwitchToExistingSession(&cfg, sessionName); err != nil {
			if errors.Is(err, tmux.ErrGracefulExit) {
				return nil
			}
			return fmt.Errorf("Failed to switch session: %w", err)
		}

		if err := tmux.KillSession(currentSession); err != nil {
			return fmt.Errorf("Failed to kill session: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
	rootCmd.PersistentFlags().BoolVarP(&killServer, "kill-server", "k", false, "Kill tmux server (rather than current session)")
}

