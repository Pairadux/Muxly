// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
) // }}}

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

func init() { // {{{
	configCmd.AddCommand(initCmd)
	// initCmd.Flags().IntP("tmux_base", "b", defaultTmuxBase, "What number your windows start ordering at.")
	// initCmd.Flags().IntP("default_depth", "d", defaultDepth, "Default depth to scan.")
	// initCmd.Flags().StringP("default_session", "D", defaultSession, "The name of the default session to fall back to.")
	// initCmd.Flags().StringArrayP("scan_dirs", "s", scanDirsToStringArray(defaultScanDirs), "A list of paths that should always be scanned.\nConcat with :int for depth.")
	// initCmd.Flags().StringArrayP("entry_dirs", "e", defaultEntryDirs, "A list of paths that are entries themselves.")
	// initCmd.Flags().StringArrayP("ignore_dirs", "i", defaultIgnoreDirs, "A list of paths that should be removed.")
	initCmd.Flags().BoolP("Defaults", "D", true /* FIXME: change to false once interactive prompt is completed */, "Accept all defaults. (No interactive prompt)")
} // }}}

func generateConfigYAML(cfg models.Config) string { // {{{
	header := `# Configuration for muxly
#
# scan_dirs: Directories to scan for projects (supports depth per directory)
#   Example: - path: ~/Dev
#            - path: ~/.config
#              depth: 2
#              alias: config
#
# entry_dirs: Additional directories always included (not scanned)
#
# ignore_dirs: Directory paths to exclude from scanning
#
# fallback_session: Session to create when killing the last tmux session
#
# tmux_base: Base index for tmux windows (0 or 1, should match your tmux.conf)
#
# default_depth: Default scanning depth for scan_dirs without explicit depth
#
# session_layout: Default layout for new tmux sessions
#   windows: List of windows to create with optional commands
#
# editor: Default editor for 'muxly config edit' (overrides $EDITOR)
#
# tmux_session_prefix: Prefix for active tmux sessions in the selector
#
# always_kill_on_last_session: Skip fallback prompt and kill server on last session

`
	yamlData, err := yaml.Marshal(cfg)
	if err != nil {
		return header + "# Error generating config: " + err.Error()
	}

	return header + string(yamlData)
} // }}}
