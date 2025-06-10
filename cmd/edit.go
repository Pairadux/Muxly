// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
) // }}}

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit [editor]",
	Short: "Edit the config file",
	Long: `Edit the config file

If you pass an optional [editor] it'll be used instead of the default $EDITOR.
You can also set the default editor in the config file that will always be used instead of $EDITOR.`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateConfig(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		editor := os.Getenv("EDITOR")
		if len(args) > 0 {
			editor = args[0]
		} else if cfgEditor := viper.GetString("editor"); cfgEditor != "" {
			editor = cfgEditor
		} else {
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
