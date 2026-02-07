package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new config file",
	Long: `Create a new config file

Creates a config file at the specified location (default location if no argument passed) if no config file exists.
Otherwise, the current config file is overwritten.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: make an interactive menu for assigning these values
		var configContent string

		useDefaults, err := cmd.Flags().GetBool("Defaults")
		if err != nil {
			return fmt.Errorf("failed to get Defaults flag: %w", err)
		}

		if useDefaults {
			configContent = generateConfigYAML(config.NewDefaultConfig())
		}

		// IDEA: before finalizing the changes, maybe diff the current file or show the config options setup and validate that they are correct

		parent := filepath.Dir(cfgFilePath)
		_ = os.MkdirAll(parent, constants.DirectoryPermissions)

		if err := os.WriteFile(cfgFilePath, []byte(configContent), constants.FilePermissions); err != nil {
			return fmt.Errorf("cannot write config: %w", err)
		}

		if verbose {
			fmt.Println("Wrote config to", cfgFilePath)
		}

		return nil
	},
}

func init() {
	configCmd.AddCommand(initCmd)
	// initCmd.Flags().IntP("tmux_base", "b", defaultTmuxBase, "What number your windows start ordering at.")
	// initCmd.Flags().IntP("default_depth", "d", defaultDepth, "Default depth to scan.")
	// initCmd.Flags().StringP("default_session", "D", defaultSession, "The name of the default session to fall back to.")
	// initCmd.Flags().StringArrayP("scan_dirs", "s", scanDirsToStringArray(defaultScanDirs), "A list of paths that should always be scanned.\nConcat with :int for depth.")
	// initCmd.Flags().StringArrayP("entry_dirs", "e", defaultEntryDirs, "A list of paths that are entries themselves.")
	// initCmd.Flags().StringArrayP("ignore_dirs", "i", defaultIgnoreDirs, "A list of paths that should be removed.")
	initCmd.Flags().BoolP("Defaults", "D", true /* FIXME: change to false once interactive prompt is completed */, "Accept all defaults. (No interactive prompt)")
}

func generateConfigYAML(cfg models.Config) string {
	header := `# Configuration for muxly
#
# scan_dirs: Directories to scan for projects (supports depth, alias, and template per directory)
#   Example: - path: ~/Dev
#            - path: ~/.config
#              depth: 2
#              alias: config
#              template: "Single Window"
#
# entry_dirs: Additional directories always included (not scanned)
#   Supports optional template assignment:
#   Example: - path: ~/special-project
#              template: "Single Window"
#
# ignore_dirs: Directory paths to exclude from scanning
#
# primary_template: The default template used for new sessions
#   name: Template name (required)
#   path: Fixed working directory (optional, uses fzf picker if omitted)
#   windows: List of windows to create with optional commands
#
# templates: Additional templates to choose from (optional)
#
# settings: General application settings
#   editor: Default editor for 'muxly config edit' (overrides $EDITOR)
#   tmux_base: Base index for tmux windows (0 or 1, should match your tmux.conf)
#   default_depth: Default scanning depth for scan_dirs without explicit depth
#   tmux_session_prefix: Prefix for active tmux sessions in the selector
#   always_kill_on_last_session: Skip prompt and kill server on last session

`
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return header + "# Error generating config: " + err.Error()
	}

	return header + string(yamlData)
}
