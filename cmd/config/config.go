package config

import "github.com/spf13/cobra"

// Cmd represents the config command
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage application configuration",
	Long:  "Manage the applications configuration\n\nUse 'config init' to create a new config file and 'config edit' to modify it.",
}
