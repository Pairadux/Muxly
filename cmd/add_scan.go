package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/utility"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
) // }}}

// addScanCmd adds a directory to scan_dirs
var addScanCmd = &cobra.Command{
	Use:     "scan [path]",
	Aliases: []string{"s"},
	Short:   "Add a directory to scan_dirs",
	Long: `Add a directory to scan_dirs in the configuration file.

Scan directories are scanned recursively to find projects. Use --depth to
control how many levels deep to scan, and --alias to add a prefix in the selector.

The path can be absolute, relative, or use tilde expansion.
Relative paths (like . or ..) will be converted to absolute paths.

Examples:
  muxly add scan ~/Dev                      # Add with default depth
  muxly add s ~/projects --depth 2          # Scan 2 levels deep
  muxly add scan ~/.config --depth 1 --alias config`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]

		resolvedPath, err := resolveInputPath(inputPath)
		if err != nil {
			return err
		}

		// Verify the path actually exists on the filesystem
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", resolvedPath)
		} else if err != nil {
			return fmt.Errorf("failed to access path %q: %w", resolvedPath, err)
		}

		// Get flags
		depth, _ := cmd.Flags().GetInt("depth")
		alias, _ := cmd.Flags().GetString("alias")

		// Check if already in scan_dirs
		for _, scanDir := range cfg.ScanDirs {
			existingPath, err := utility.ResolvePath(scanDir.Path)
			if err != nil {
				continue
			}
			if existingPath == resolvedPath {
				fmt.Printf("Path %q is already in scan_dirs\n", resolvedPath)
				return nil
			}
		}

		// Create new ScanDir entry
		newScanDir := models.ScanDir{
			Path: resolvedPath,
		}

		// Only set depth if explicitly provided (nil means use default_depth)
		if cmd.Flags().Changed("depth") {
			newScanDir.Depth = &depth
		}

		// Only set alias if provided
		if alias != "" {
			newScanDir.Alias = alias
		}

		// Add to scan_dirs and write config using viper
		updatedScanDirs := append(cfg.ScanDirs, newScanDir)
		viper.Set("scan_dirs", updatedScanDirs)

		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Added %q to scan_dirs\n", resolvedPath)
		return nil
	},
}

func init() {
	addCmd.AddCommand(addScanCmd)

	// Flags for scan subcommand
	addScanCmd.Flags().IntP("depth", "d", 0, "Scanning depth (0 = use default_depth)")
	addScanCmd.Flags().StringP("alias", "a", "", "Alias prefix for the directory in selector")
}
