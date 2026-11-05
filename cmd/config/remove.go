package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Pairadux/muxly/internal/forms"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// removeCmd represents the remove command
var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove directories from configuration",
	Long: `Remove directories from entry_dirs or scan_dirs in the configuration file.

Use subcommands to specify where to remove the directory from:
  entry (e) - Remove from entry_dirs
  scan (s)  - Remove from scan_dirs

Examples:
  muxly config remove entry ~/Dev/my-project
  muxly config remove e .
  muxly config remove scan ~/Dev
  muxly config remove s ~/projects`,
}

func init() {
	Cmd.AddCommand(removeCmd)
}

// shouldDeleteMuxlyFile decides what happens to a directory's .muxly file when its entry
// is removed. It reports false if there isn't one.
//
// Behavior depends on flags and environment:
//   - --keep flag: Keeps .muxly file
//   - --delete flag: Deletes .muxly file
//   - No flags + interactive (TTY): Prompts user with a confirmation form
//   - No flags + non-interactive: Returns error requiring explicit flag choice
//
// This prevents accidental .muxly deletion in scripts while providing a smooth
// interactive experience. Returns error if --keep and --delete are both specified.
func shouldDeleteMuxlyFile(cmd *cobra.Command, dir string) (bool, error) {
	if _, err := os.Stat(filepath.Join(dir, ".muxly")); err != nil {
		// No .muxly file, nothing to do
		return false, nil
	}

	keep, _ := cmd.Flags().GetBool("keep")
	deleteMuxly, _ := cmd.Flags().GetBool("delete")

	// Check for conflicting flags
	if keep && deleteMuxly {
		return false, fmt.Errorf("cannot use both --keep and --delete flags")
	}

	if keep || deleteMuxly {
		return deleteMuxly, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return false, fmt.Errorf(".muxly file found in %s\nFor non-interactive use, specify: --keep (-k) to keep or --delete (-d) to delete", dir)
	}

	var shouldDelete bool
	form := forms.ConfirmationForm(
		"Delete .muxly file?",
		fmt.Sprintf("Found .muxly file in %s", dir),
		&shouldDelete,
	)
	if err := form.Run(); err != nil {
		return false, fmt.Errorf("failed to run confirmation form: %w", err)
	}
	return shouldDelete, nil
}

// deleteMuxlyFile removes a directory's .muxly file. By now its entry is gone from the
// config, so failing here only earns a warning.
func deleteMuxlyFile(dir string) {
	if err := os.Remove(filepath.Join(dir, ".muxly")); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to delete .muxly file: %v\n", err)
		return
	}
	fmt.Println("Deleted .muxly file")
}
