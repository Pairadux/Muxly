// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"

	"github.com/Pairadux/tms/internal/fzf"
	"github.com/Pairadux/tms/internal/tmux"

	"github.com/spf13/cobra"
) // ) // }}}

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill the current session and replace with another",
	Long: `Kill the current session and replace with another

A picker list of alternative sessions will be displayed to switch the current session.
If there are no other sessions however, the default sessions configured in the config file will be used.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := tmux.ValidateTmuxAvailable(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := validateConfig(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		currentSession := tmux.GetCurrentTmuxSession()

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			// FIXME: if empty, need to fallback to default
			sessions := tmux.GetSessionsExceptCurrent(currentSession)

			var err error
			choiceStr, err = fzf.SelectWithFzf(sessions)
			if err != nil {
				if err.Error() == "user cancelled" {
					os.Exit(0)
				}
				cobra.CheckErr(err)
			}

			if choiceStr == "" {
				os.Exit(0)
			}
		}
		sessionName := choiceStr
		if err := tmux.SwitchToExistingSession(sessionName); err != nil {
			fmt.Fprintf(os.Stderr, "failed to switch session: %v\n", err)
			os.Exit(1)
		}

		if err := tmux.KillSession(currentSession); err != nil {
			fmt.Fprintf(os.Stderr, "failed to kill session: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}

