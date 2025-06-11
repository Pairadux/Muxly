// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"

	"github.com/spf13/cobra"
) // }}}

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill",
	Short: "Kill the current session and replace with another",
	Long: `Kill the current session and replace with another

A picker list of alternative sessions will be displayed to switch the current session.
If there are no other sessions however, the default sessions configured in the config file will be used.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("kill called")
	},
}

func init() {
	rootCmd.AddCommand(killCmd)
}
