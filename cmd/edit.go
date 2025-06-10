// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
) // }}}

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit [editor]",
	Short: "Edit the config file",
	Long: `Edit the config file

If you pass an optional [editor] it'll be used instead of the default $EDITOR.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		editor := os.Getenv("EDITOR")
		if len(args) > 0 {
			editor = args[0]
		}
		if editor == "" {
			editor = "vi"
		}
		editCmd := exec.Command(editor, cfgFilePath)
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr
		if err := editCmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	},
}

func init() {
	configCmd.AddCommand(editCmd)
}
