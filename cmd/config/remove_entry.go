package config

import (
	"errors"
	"fmt"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/forms"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/utility"
	"github.com/spf13/cobra"
)

// removeEntryCmd removes a directory from entry_dirs
var removeEntryCmd = &cobra.Command{
	Use:     "entry [path]",
	Aliases: []string{"e"},
	Short:   "Remove a directory from entry_dirs",
	Long: `Remove a directory from entry_dirs in the configuration file.

The path can be absolute, start with ~, be relative to the current folder, or be
exactly as 'muxly config list' shows it.

If a .muxly file exists in the directory, you will be prompted to delete it
in interactive mode. In non-interactive environments, you must explicitly
choose to keep (-k) or delete (-d) the .muxly file.

Examples:
  muxly config remove entry .                      # Remove current directory (interactive)
  muxly config remove e ~/Dev/my-project           # Remove project (interactive)
  muxly config remove entry -k /path/to/directory  # Remove, keep .muxly (non-interactive)
  muxly config remove entry -d ~/old-project       # Remove, delete .muxly (non-interactive)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		resolvedPath, err := utility.ResolveArgumentPath(args[0])
		if err != nil {
			return err
		}

		idx := findConfigured(state.Cfg.EntryDirs, func(ed models.EntryDir) string { return ed.Path }, args[0], resolvedPath)
		if idx == -1 {
			return fmt.Errorf("path %q is not in entry_dirs", resolvedPath)
		}
		written := state.Cfg.EntryDirs[idx].Path

		// The entry's own path says where its .muxly file would be, since it may not be what
		// was typed. An entry that doesn't resolve has no directory to look in.
		deleteMuxly := false
		dir, resolveErr := utility.ResolvePath(written)
		if resolveErr == nil {
			deleteMuxly, err = shouldDeleteMuxlyFile(cmd, dir)
			if errors.Is(err, forms.ErrCancelled) {
				return nil
			}
			if err != nil {
				return err
			}
		}

		// The .muxly file goes last, so a config that can't be changed costs nothing.
		if err := config.RemoveEntryDir(state.CfgFilePath, idx); err != nil {
			return err
		}
		if deleteMuxly {
			deleteMuxlyFile(dir)
		}

		fmt.Printf("Removed %q from entry_dirs\n", written)
		return nil
	},
}

func init() {
	removeCmd.AddCommand(removeEntryCmd)
	removeEntryCmd.Flags().BoolP("keep", "k", false, "Keep .muxly file (for non-interactive use)")
	removeEntryCmd.Flags().BoolP("delete", "d", false, "Delete .muxly file (for non-interactive use)")
}
