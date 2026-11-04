package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new config file",
	Long: `Create a new config file

Writes a default config to the config path and prints where it wrote it.
If a file already exists there, it is left alone unless you pass --force.`,
	Annotations: map[string]string{
		constants.AnnotationSkipConfigCheck: "true",
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		force, err := cmd.Flags().GetBool("force")
		if err != nil {
			return fmt.Errorf("reading --force: %w", err)
		}

		content, err := generateConfigYAML(config.NewDefaultConfig())
		if err != nil {
			return fmt.Errorf("generating config: %w", err)
		}

		if err := writeNewConfig(state.CfgFilePath, content, force); err != nil {
			return err
		}

		fmt.Println("Wrote config to", state.CfgFilePath)
		return nil
	},
}

func init() {
	Cmd.AddCommand(initCmd)
	initCmd.Flags().BoolP("force", "f", false, "Overwrite an existing config file")
}

// writeNewConfig writes content to path, creating its directory if needed.
// Unless force is set, it refuses to replace a file that's already there.
func writeNewConfig(path, content string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config file already exists: %s\nUse --force to overwrite it, or 'muxly config edit' to change it", path)
		} else if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("checking for an existing config: %w", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), constants.DirectoryPermissions); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(content), constants.FilePermissions); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}
	return nil
}

func generateConfigYAML(cfg models.Config) (string, error) {
	header := `# Configuration for muxly
#
# scan_dirs: Directories to scan for projects (supports depth, alias, and template per directory)
#   Example: - path: ~/Dev
#            - path: ~/.config
#              depth: 2
#              alias: config
#              template: minimal
#
# entry_dirs: Additional directories always included (not scanned)
#   Supports optional template assignment:
#   Example: - path: ~/special-project
#              template: minimal
#
# ignore_dirs: Directories to exclude from scanning (additive)
#   .git and node_modules are always ignored and do not need to be listed.
#   Entries here are added on top of those built-in filters.
#   Bare names (e.g. "target") match any directory with that name at any depth
#   Paths (e.g. "~/projects/old") match only that specific resolved directory
#
# templates: Session templates (one must have default: true)
#   name: Short identifier used as the tmux session name (required)
#   label: Human-readable display name shown in the muxly create TUI (optional)
#   default: Mark exactly one template as the default (required on one template)
#   path: Fixed working directory (optional, uses fzf picker if omitted)
#   windows: List of windows to create with optional commands
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
		return "", err
	}

	return header + string(yamlData), nil
}
