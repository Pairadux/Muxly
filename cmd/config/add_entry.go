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

// addEntryCmd adds a directory to entry_dirs
var addEntryCmd = &cobra.Command{
	Use:     "entry [path]",
	Aliases: []string{"e"},
	Short:   "Add a directory to entry_dirs",
	Long: `Add a directory to entry_dirs in the configuration file.

Entry directories are included directly without scanning. Use this for
specific directories you want to access, not entire directory trees.

The path can be absolute, start with ~, or be relative to the current folder.
It's written to the config starting with ~ if it's in your home directory.
The rest of the file, including comments, stays as it is.

Examples:
  muxly config add entry .                    # Add current directory
  muxly config add e ~/Dev/my-project         # Add a specific project
  muxly config add entry /path/to/directory   # Add absolute path`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolvedPath, err := utility.ResolveArgumentPath(args[0])
		if err != nil {
			return err
		}

		if err := utility.RequireDirectory(resolvedPath); err != nil {
			return err
		}

		// Check if this path would already be discovered by a scan_dir
		// This prevents redundant configuration
		if matchedScanDir, depth, found := wouldBeFoundByScanDirs(resolvedPath, state.Cfg.ScanDirs); found {
			return fmt.Errorf("path %q would already be found by scan_dir %s (depth: %d)\nNo need to add it to entry_dirs", resolvedPath, matchedScanDir.String(), depth)
		}

		if slices.ContainsFunc(state.Cfg.EntryDirs, func(ed models.EntryDir) bool {
			return resolvesTo(ed.Path, resolvedPath)
		}) {
			fmt.Printf("Path %q is already in entry_dirs\n", resolvedPath)
			return nil
		}

		written, err := pathForConfig(resolvedPath)
		if err != nil {
			return err
		}
		if err := config.AddEntryDir(state.CfgFilePath, models.EntryDir{Path: written}); err != nil {
			return err
		}

		fmt.Printf("Added %q to entry_dirs\n", written)
		return nil
	},
}

func init() {
	addCmd.AddCommand(addEntryCmd)
}
