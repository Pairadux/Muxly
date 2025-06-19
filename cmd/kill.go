// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"

	"github.com/Pairadux/Tmux-Sessionizer/internal/fzf"
	"github.com/Pairadux/Tmux-Sessionizer/internal/tmux"

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
			// IDEA: consider prompting the user to kill the running tmux session if one is available
			return fmt.Errorf("Not in Tmux, use 'tms' to get started.")
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			sessions := tmux.GetSessionsExceptCurrent(currentSession)

			if len(sessions) == 0 {
				// IDEA: maybe rather than just immediately dropping back to fallback, prompt user to fallback
				// If "no" then kill server
				if err := tmux.CreateAndSwitchToFallbackSession(&cfg); err != nil {
					return fmt.Errorf("Failed to create default session: %w", err)
				}
				if err := tmux.KillSession(currentSession); err != nil {
					return fmt.Errorf("Failed to kill session: %w", err)
				}

				return nil
			}

			var err error
			choiceStr, err = fzf.SelectWithFzf(sessions)
			if err != nil {
				if err.Error() == "user cancelled" {
					return nil
				}
				cobra.CheckErr(err)
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
