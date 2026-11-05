package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/spf13/cobra"
)

// editCmd represents the edit command
var editCmd = &cobra.Command{
	Use:   "edit [editor]",
	Short: "Edit the config file",
	Long: `Edit the config file

The editor is the first of these that's set: the [editor] you pass, $MUXLY_EDITOR,
settings.editor in the config, $VISUAL, $EDITOR, then vi. It can include arguments,
such as "code --wait".`,
	Args: cobra.MaximumNArgs(1),
	Annotations: map[string]string{
		constants.AnnotationSkipConfigCheck: "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := runEditor(pickEditor(args), state.CfgFilePath); err != nil {
			return fmt.Errorf("editor exited with error: %w", err)
		}

		// Validate the edited config file
		if _, err := config.ValidateConfigFile(state.CfgFilePath); err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: Config validation failed: %v\n", err)
			fmt.Fprintln(os.Stderr, "Please fix the issues above or your config may not work correctly.")
			return fmt.Errorf("config validation failed")
		}

		fmt.Fprintln(os.Stderr, "Config file validated successfully")
		return nil
	},
}

func init() {
	Cmd.AddCommand(editCmd)
}

func pickEditor(args []string) string {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return args[0]
	}
	return config.Editor(state.Cfg.Settings.Editor)
}

// runEditor opens path in an editor command. It goes through a shell, as git does, because
// the command can hold arguments. The path goes in as "$1", so nothing in it needs quoting.
func runEditor(editor, path string) error {
	cmd := exec.Command("sh", "-c", editor+` "$1"`, "sh", path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
