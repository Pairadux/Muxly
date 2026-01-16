package cmd

// IMPORTS {{{
import (
	"fmt"
	"slices"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
) // }}}

// removeEntryCmd removes a directory from entry_dirs
var removeEntryCmd = &cobra.Command{
	Use:     "entry [path]",
	Aliases: []string{"e"},
	Short:   "Remove a directory from entry_dirs",
	Long: `Remove a directory from entry_dirs in the configuration file.

The path can be absolute, relative, or use tilde expansion.
Relative paths (like . or ..) will be converted to absolute paths.

If a .muxly file exists in the directory, you will be prompted to delete it
in interactive mode. In non-interactive environments, you must explicitly
choose to keep (-k) or delete (-d) the .muxly file.

Examples:
  muxly remove entry .                      # Remove current directory (interactive)
  muxly remove e ~/Dev/my-project           # Remove project (interactive)
  muxly remove entry -k /path/to/directory  # Remove, keep .muxly (non-interactive)
  muxly remove entry -d ~/old-project       # Remove, delete .muxly (non-interactive)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]

		resolvedPath, err := resolveInputPath(inputPath)
		if err != nil {
			return err
		}

		// Check if path exists in entry_dirs
		idx := slices.Index(cfg.EntryDirs, resolvedPath)
		if idx == -1 {
			return fmt.Errorf("path %q is not in entry_dirs", resolvedPath)
		}

		// Handle .muxly file deletion
		if err := handleMuxlyFile(cmd, resolvedPath); err != nil {
			return err
		}

		// Remove from entry_dirs by creating a new slice without the element
		updatedEntryDirs := slices.Delete(cfg.EntryDirs, idx, idx+1)
		viper.Set("entry_dirs", updatedEntryDirs)

		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Removed %q from entry_dirs\n", resolvedPath)
		return nil
	},
}

func init() {
	removeCmd.AddCommand(removeEntryCmd)
	removeEntryCmd.Flags().BoolP("keep", "k", false, "Keep .muxly file (for non-interactive use)")
	removeEntryCmd.Flags().BoolP("delete", "d", false, "Delete .muxly file (for non-interactive use)")
}
