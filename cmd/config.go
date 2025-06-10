// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

import "github.com/spf13/cobra"

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage application configuration",
	Long: "Manage the applications configuration\n\nUse 'config init' to create a new config file and 'config edit' to modify it.",
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("config called")
	// },
}

func init() {
	rootCmd.AddCommand(configCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
