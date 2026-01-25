package cmd


import (
	"fmt"
	"os"
	"os/exec"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/spf13/cobra"
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit [editor]",
	Short: "Edit the config file",
	Long: `Edit the config file

If you pass an optional [editor] it'll be used instead of the default $EDITOR.
You can also set the default editor in the config file that will always be used instead of $EDITOR.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := pickEditor(args)

		editCmd := exec.Command(editor, cfgFilePath)
		editCmd.Stdin = os.Stdin
		editCmd.Stdout = os.Stdout
		editCmd.Stderr = os.Stderr
		if err := editCmd.Run(); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		// Validate the edited config file
		if _, err := config.ValidateConfigFile(cfgFilePath); err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: Config validation failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "Please fix the issues above or your config may not work correctly.")
			return fmt.Errorf("config validation failed")
		}

		fmt.Fprintln(os.Stderr, "Config file validated successfully")
		return nil
	},
}

func init() {
	configCmd.AddCommand(editCmd)
}

func pickEditor(args []string) string {
	// Precedence: CLI arg > cfg.Settings.Editor (from MUXLY_EDITOR/EDITOR env or config file) > default
	if len(args) > 0 {
		return args[0]
	}

	if cfg.Settings.Editor != "" {
		return cfg.Settings.Editor
	}

	return config.DefaultEditor
}
