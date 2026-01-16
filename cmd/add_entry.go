package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
) // }}}

// addEntryCmd adds a directory to entry_dirs
var addEntryCmd = &cobra.Command{
	Use:     "entry [path]",
	Aliases: []string{"e"},
	Short:   "Add a directory to entry_dirs",
	Long: `Add a directory to entry_dirs in the configuration file.

Entry directories are included directly without scanning. Use this for
specific directories you want to access, not entire directory trees.

The path can be absolute, relative, or use tilde expansion.
Relative paths (like . or ..) will be converted to absolute paths.

Examples:
  muxly add entry .                    # Add current directory
  muxly add e ~/Dev/my-project         # Add a specific project
  muxly add entry /path/to/directory   # Add absolute path`,
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

		// Check if this path would already be discovered by a scan_dir
		// This prevents redundant configuration
		if matchedScanDir, depth, found := wouldBeFoundByScanDirs(resolvedPath, cfg.ScanDirs); found {
			return fmt.Errorf("path %q would already be found by scan_dir %s (depth: %d)\nNo need to add it to entry_dirs", resolvedPath, matchedScanDir.String(), depth)
		}

		// Check if already in entry_dirs to avoid duplicates
		if slices.Contains(cfg.EntryDirs, resolvedPath) {
			fmt.Printf("Path %q is already in entry_dirs\n", resolvedPath)
			return nil
		}

		// Add to entry_dirs and write config using viper
		updatedEntryDirs := append(cfg.EntryDirs, resolvedPath)
		viper.Set("entry_dirs", updatedEntryDirs)

		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Added %q to entry_dirs\n", resolvedPath)
		return nil
	},
}

func init() {
	addCmd.AddCommand(addEntryCmd)
}
