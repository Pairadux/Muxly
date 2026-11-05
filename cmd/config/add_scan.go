package config

import (
	"fmt"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/utility"
	"github.com/spf13/cobra"
)

// addScanCmd adds a directory to scan_dirs
var addScanCmd = &cobra.Command{
	Use:     "scan [path]",
	Aliases: []string{"s"},
	Short:   "Add a directory to scan_dirs",
	Long: `Add a directory to scan_dirs in the configuration file.

Scan directories are scanned recursively to find projects. Use --depth to
control how many levels deep to scan, and --alias to add a prefix in the selector.

The path can be absolute, start with ~, or be relative to the current folder.
It's written to the config starting with ~ if it's in your home directory.
The rest of the file, including comments, stays as it is.

Examples:
  muxly config add scan ~/Dev                      # Add with default depth
  muxly config add s ~/projects --depth 2          # Scan 2 levels deep
  muxly config add scan ~/.config --depth 1 --alias config`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolvedPath, err := utility.ResolveArgumentPath(args[0])
		if err != nil {
			return err
		}

		if err := utility.RequireDirectory(resolvedPath); err != nil {
			return err
		}

		// Get flags
		depth, _ := cmd.Flags().GetInt("depth")
		alias, _ := cmd.Flags().GetString("alias")
		if cmd.Flags().Changed("depth") && depth < 1 {
			return fmt.Errorf("--depth must be at least 1, got %d", depth)
		}

		// Check if already in scan_dirs
		for _, scanDir := range state.Cfg.ScanDirs {
			if resolvesTo(scanDir.Path, resolvedPath) {
				fmt.Printf("Path %q is already in scan_dirs\n", resolvedPath)
				return nil
			}
		}

		written, err := pathForConfig(resolvedPath)
		if err != nil {
			return err
		}

		// Create new ScanDir entry
		newScanDir := models.ScanDir{
			Path: written,
		}

		// Only set depth if explicitly provided (nil means use default_depth)
		if cmd.Flags().Changed("depth") {
			newScanDir.Depth = &depth
		}

		// Only set alias if provided
		if alias != "" {
			newScanDir.Alias = alias
		}

		if err := config.AddScanDir(state.CfgFilePath, newScanDir); err != nil {
			return err
		}

		fmt.Printf("Added %q to scan_dirs\n", written)
		return nil
	},
}

func init() {
	addCmd.AddCommand(addScanCmd)

	// Flags for scan subcommand
	addScanCmd.Flags().IntP("depth", "d", 0, "Scanning depth, at least 1 (uses default_depth if not set)")
	addScanCmd.Flags().StringP("alias", "a", "", "Alias prefix for the directory in selector")
}
