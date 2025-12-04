// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Pairadux/muxly/internal/forms"
	"github.com/Pairadux/muxly/internal/utility"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
) // }}}

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove [path]",
	Short: "Remove a directory from entry_dirs",
	Long: `Remove a directory from entry_dirs in the configuration file.

The path can be absolute, relative, or use tilde expansion.
Relative paths (like . or ..) will be converted to absolute paths.

If a .muxly file exists in the directory, you will be prompted to delete it
in interactive mode. In non-interactive environments, you must explicitly
choose to keep (-k) or delete (-d) the .muxly file.

Examples:
  muxly remove .                      # Remove current directory (interactive)
  muxly remove ~/Dev/my-project       # Remove project (interactive)
  muxly remove -k /path/to/directory  # Remove, keep .muxly (non-interactive)
  muxly remove -d ~/old-project       # Remove, delete .muxly (non-interactive)`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]

		var resolvedPath string
		var err error

		// Handle relative paths like "." and ".."
		// filepath.Abs converts them to absolute paths based on current working directory
		if inputPath == "." || inputPath == ".." {
			absPath, err := filepath.Abs(inputPath)
			if err != nil {
				return fmt.Errorf("failed to resolve relative path %q: %w", inputPath, err)
			}
			// Still pass through ResolvePath to handle any env vars and path cleaning
			resolvedPath, err = utility.ResolvePath(absPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path %q: %w", absPath, err)
			}
		} else {
			// For absolute paths, tilde paths, or paths with env vars
			resolvedPath, err = utility.ResolvePath(inputPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path %q: %w", inputPath, err)
			}
		}

		// Check if path exists in entry_dirs
		idx := slices.Index(cfg.EntryDirs, resolvedPath)
		if idx == -1 {
			return fmt.Errorf("path %q is not in entry_dirs", resolvedPath)
		}

		// Check for .muxly file and handle deletion
		muxlyFile := filepath.Join(resolvedPath, ".muxly")
		if _, err := os.Stat(muxlyFile); err == nil {
			keep, _ := cmd.Flags().GetBool("keep")
			deleteMuxly, _ := cmd.Flags().GetBool("delete")

			// Check for conflicting flags
			if keep && deleteMuxly {
				return fmt.Errorf("cannot use both --keep and --delete flags")
			}

			var shouldDelete bool

			// Determine behavior based on flags and environment
			if keep {
				shouldDelete = false
			} else if deleteMuxly {
				shouldDelete = true
			} else {
				// No flags specified - check if interactive
				if term.IsTerminal(int(os.Stdin.Fd())) {
					// Interactive: prompt user
					form := forms.ConfirmationForm(
						"Delete .muxly file?",
						fmt.Sprintf("Found .muxly file in %s", resolvedPath),
						&shouldDelete,
					)

					if err := form.Run(); err != nil {
						return fmt.Errorf("failed to run confirmation form: %w", err)
					}
				} else {
					// Non-interactive without flags: error
					return fmt.Errorf(".muxly file found in %s\nFor non-interactive use, specify: --keep (-k) to keep or --delete (-d) to delete", resolvedPath)
				}
			}

			if shouldDelete {
				if err := os.Remove(muxlyFile); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to delete .muxly file: %v\n", err)
				} else {
					fmt.Println("Deleted .muxly file")
				}
			}
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
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags().BoolP("keep", "k", false, "Keep .muxly file (for non-interactive use)")
	removeCmd.Flags().BoolP("delete", "d", false, "Delete .muxly file (for non-interactive use)")
}
