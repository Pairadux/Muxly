// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"

	"github.com/Pairadux/tms/internal/fzf"
	"github.com/Pairadux/tms/internal/tmux"
	"github.com/Pairadux/tms/internal/utility"

	"github.com/spf13/cobra"
) // }}}

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch [SESSION]",
	Short: "Switch to an active session",
	Long:  "Switch to an active session\n\nDisplays a fzf picker list of active sessions.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := tmux.ValidateTmuxAvailable(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := utility.ValidateConfig(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		currentSession := tmux.GetCurrentTmuxSession()

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			// FIXME: if empty need to error out or something
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
	},
}

func init() { 
	rootCmd.AddCommand(switchCmd)
} 

