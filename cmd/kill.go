// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"

	"github.com/Pairadux/muxly/internal/forms"
	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/tmux"

	"github.com/spf13/cobra"
) // }}}

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill the current session and replace with another",
	Long: `Kill the current session and replace with another

A picker list of alternative sessions will be displayed to switch the current session.
If there are no other sessions however, the default sessions configured in the config file will be used.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		currentSession := tmux.GetCurrentTmuxSession()

		if currentSession == "" {
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
				// If configured to always kill on last session, skip the prompt
				if cfg.AlwaysKillOnLastSession {
					if err := tmux.KillServer(); err != nil {
						return fmt.Errorf("failed to kill tmux server: %w", err)
					}
					fmt.Println("tmux server killed.")
					return nil
				}

				var createFallback bool
				form := forms.ConfirmationForm("Create default session?", "Declining will kill the tmux server.", &createFallback)

				if err := form.Run(); err != nil {
					return fmt.Errorf("failed to run confirmation form: %w", err)
				}

				if createFallback {
					if err := tmux.CreateAndSwitchToFallbackSession(&cfg); err != nil {
						return fmt.Errorf("Failed to create default session: %w", err)
					}
					if err := tmux.KillSession(currentSession); err != nil {
						return fmt.Errorf("Failed to kill session: %w", err)
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
}
