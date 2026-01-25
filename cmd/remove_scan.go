package cmd


import (
	"fmt"

	"github.com/Pairadux/muxly/internal/utility"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// removeScanCmd removes a directory from scan_dirs
var removeScanCmd = &cobra.Command{
	Use:     "scan [path]",
	Aliases: []string{"s"},
	Short:   "Remove a directory from scan_dirs",
	Long: `Remove a directory from scan_dirs in the configuration file.

The path can be absolute, relative, or use tilde expansion.
Relative paths (like . or ..) will be converted to absolute paths.

Examples:
  muxly remove scan ~/Dev          # Remove from scan_dirs
  muxly remove s ~/projects        # Short form
  muxly remove scan ~/.config      # Remove config directory`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]

		resolvedPath, err := resolveInputPath(inputPath)
		if err != nil {
			return err
		}

		// Find the scan_dir entry
		foundIdx := -1
		for i, scanDir := range cfg.ScanDirs {
			existingPath, err := utility.ResolvePath(scanDir.Path)
			if err != nil {
				continue
			}
			if existingPath == resolvedPath {
				foundIdx = i
				break
			}
		}

		if foundIdx == -1 {
			return fmt.Errorf("path %q is not in scan_dirs", resolvedPath)
		}

		// Remove from scan_dirs
		updatedScanDirs := append(cfg.ScanDirs[:foundIdx], cfg.ScanDirs[foundIdx+1:]...)
		viper.Set("scan_dirs", updatedScanDirs)

		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Removed %q from scan_dirs\n", resolvedPath)
		return nil
	},
}

func init() {
	removeCmd.AddCommand(removeScanCmd)
}
