// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Pairadux/tms/internal/utility"
	"github.com/spf13/cobra"
) // }}}

const DefaultEditor = "vi"

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit [editor]",
	Short: "Edit the config file",
	Long: `Edit the config file

If you pass an optional [editor] it'll be used instead of the default $EDITOR.
You can also set the default editor in the config file that will always be used instead of $EDITOR.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := utility.ValidateConfig(&cfg); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}

		editor := pickEditor(args)

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

func pickEditor(args []string) string {
	env := os.Getenv("EDITOR")

	switch {
	case len(args) > 0:
		return args[0]
	case cfg.Editor != "":
		return cfg.Editor
	case env != "":
		return env
	default:
		return DefaultEditor
	}
}
