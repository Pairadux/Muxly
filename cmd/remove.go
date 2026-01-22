// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Pairadux/muxly/internal/forms"
	"github.com/spf13/cobra"
	"golang.org/x/term"
) // }}}

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove directories from configuration",
	Long: `Remove directories from entry_dirs or scan_dirs in the configuration file.

Use subcommands to specify where to remove the directory from:
  entry (e) - Remove from entry_dirs
  scan (s)  - Remove from scan_dirs

Examples:
  muxly remove entry ~/Dev/my-project
  muxly remove e .
  muxly remove scan ~/Dev
  muxly remove s ~/projects`,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

// handleMuxlyFile manages .muxly file deletion when removing an entry directory.
//
// Behavior depends on flags and environment:
//   - --keep flag: Keeps .muxly file
//   - --delete flag: Deletes .muxly file
//   - No flags + interactive (TTY): Prompts user with a confirmation form
//   - No flags + non-interactive: Returns error requiring explicit flag choice
//
// This prevents accidental .muxly deletion in scripts while providing a smooth
// interactive experience. Returns error if --keep and --delete are both specified.
func handleMuxlyFile(cmd *cobra.Command, resolvedPath string) error {
	muxlyFile := filepath.Join(resolvedPath, ".muxly")
	if _, err := os.Stat(muxlyFile); err != nil {
		// No .muxly file, nothing to do
		return nil
	}

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

	return nil
}
