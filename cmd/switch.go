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
) // }}}

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch to an active session",
	Long:  `Switch to an active session`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := tmux.ValidateTmuxAvailable(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := validateConfig(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			sessions := tmux.GetTmuxSessionNames()
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

func init() { // {{{
	rootCmd.AddCommand(switchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// switchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// switchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
} // }}}
