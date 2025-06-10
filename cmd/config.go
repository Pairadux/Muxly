// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

import "github.com/spf13/cobra"

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage application configuration",
	Long:  "Manage the applications configuration\n\nUse 'config init' to create a new config file and 'config edit' to modify it.",
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("config called")
	// },
}

func init() {
	rootCmd.AddCommand(configCmd)
}
