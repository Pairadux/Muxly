// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"bufio"
	"fmt"
	"os"

	"github.com/Pairadux/Tmux-Sessionizer/internal/fzf"
	"github.com/Pairadux/Tmux-Sessionizer/internal/tmux"

	"github.com/spf13/cobra"
) // }}}

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch [SESSION]",
	Short: "Switch to an active session",
	Long: `Switch to an active session

Displays a fzf picker list of active sessions.
If no other sessions found, exit.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		currentSession := tmux.GetCurrentTmuxSession()

		if currentSession == "" {
			return fmt.Errorf("Not in Tmux, use 'tms' to get started.")
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			sessions := tmux.GetSessionsExceptCurrent(currentSession)

			if len(sessions) == 0 {
				fmt.Println("No other sessions available. Use 'tms' to start a new session.")
				fmt.Print("Press Enter to exit...")
				// REFACTOR: Consider using a more user-friendly way to pause execution
				bufio.NewReader(os.Stdin).ReadBytes('\n')

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

		return nil
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
