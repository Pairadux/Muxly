package config

import "github.com/spf13/cobra"

// listCmd lists all scan_dirs and entry_dirs in config
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List all scan_dirs and entry_dirs present in config",
	Long: `List all scan_dirs and entry_dirs currently configured.

Scan directories are traversed up to their configured depth to discover
projects, while entry directories are included directly without scanning.
This shows every directory of both kinds present in the configuration file.

Examples:
  muxly config list   # List all configured directories
  muxly config l      # Same, using the alias`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)
}
