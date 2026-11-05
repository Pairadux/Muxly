package config

import (
	"fmt"
	"slices"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/utility"
	"github.com/spf13/cobra"
)

// removeScanCmd removes a directory from scan_dirs
var removeScanCmd = &cobra.Command{
	Use:     "scan [path]",
	Aliases: []string{"s"},
	Short:   "Remove a directory from scan_dirs",
	Long: `Remove a directory from scan_dirs in the configuration file.

The path can be absolute, start with ~, be relative to the current folder, or be
exactly as 'muxly config list' shows it.

Examples:
  muxly config remove scan ~/Dev          # Remove from scan_dirs
  muxly config remove s ~/projects        # Short form
  muxly config remove scan ~/.config      # Remove config directory`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolvedPath, err := utility.ResolveArgumentPath(args[0])
		if err != nil {
			return err
		}

		idx := slices.IndexFunc(state.Cfg.ScanDirs, func(sd models.ScanDir) bool {
			return sd.Path == args[0] || resolvesTo(sd.Path, resolvedPath)
		})
		if idx == -1 {
			return fmt.Errorf("path %q is not in scan_dirs", resolvedPath)
		}

		if err := config.RemoveScanDir(state.CfgFilePath, idx); err != nil {
			return err
		}

		fmt.Printf("Removed %q from scan_dirs\n", state.Cfg.ScanDirs[idx].Path)
		return nil
	},
}

func init() {
	removeCmd.AddCommand(removeScanCmd)
}
