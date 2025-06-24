// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Pairadux/Tmux-Sessionizer/internal/models"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
) // }}}

// Default values - defined once and used everywhere
var (
	defaultScanDirs = []models.ScanDir{
		{Path: "~/Dev", Depth: nil, Alias: ""},
		{Path: "~/.dotfiles/dot_config", Depth: nil, Alias: ""},
	}
	defaultEntryDirs  = []string{"~/Documents", "~/Cloud"}
	defaultIgnoreDirs = []string{"~/Dev/_practice", "~/Dev/_archive"}
	defaultTmuxBase   = 1
	defaultDepth      = 1
	fallbackSession   = models.Session{
		Name: "Default",
		Path: "~/",
		Layout: models.SessionLayout{
			Windows: []models.Window{
				{Name: "window", Cmd: ""},
			},
		},
	}
	defaultSessionLayout = models.SessionLayout{
		Windows: []models.Window{
			{Name: "edit", Cmd: "nvim"},
			{Name: "term", Cmd: ""},
		},
	}
	defaultEditor            = "vi"
	defaultTmuxSessionPrefix = "[TMUX] "
	// TODO: add config option for "use-absolute-path"
	// This would change the entries from using the basename to using the resolved absolute path in the fzf selector
	// TODO: add a config option to remove current session from list of options 
	// Might would help with the duplicate problem, especially in conjuction with absolute path config option
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
		scanDirs := defaultScanDirs
		entryDirs := defaultEntryDirs
		ignoreDirs := defaultIgnoreDirs
		tmuxBase := defaultTmuxBase
		depth := defaultDepth
		session := fallbackSession
		sessionLayout := defaultSessionLayout
		editor := defaultEditor
		tmuxSessionPrefix := defaultTmuxSessionPrefix

		configContent := generateConfigYAML(models.Config{
			ScanDirs:          scanDirs,
			EntryDirs:         entryDirs,
			IgnoreDirs:        ignoreDirs,
			FallbackSession:   session,
			TmuxBase:          tmuxBase,
			DefaultDepth:      depth,
			SessionLayout:     sessionLayout,
			Editor:            editor,
			TmuxSessionPrefix: tmuxSessionPrefix,
		})

		parent := filepath.Dir(cfgFilePath)
		_ = os.MkdirAll(parent, 0o755)

		if err := os.WriteFile(cfgFilePath, []byte(configContent), 0o644); err != nil {
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
} // }}}

func generateConfigYAML(params models.Config) string { // {{{
	var b strings.Builder

	b.WriteString("# Configuration for Tmux Session Manager\n\n")

	// Scan directories
	b.WriteString("# Directories to scan for projects\n")
	b.WriteString("# Each entry can be a simple path or include depth:\n")
	b.WriteString("#   - path: ~/\n")
	b.WriteString("#     depth: 3\n")
	scanDirsYAML, _ := yaml.Marshal(map[string][]models.ScanDir{"scan_dirs": params.ScanDirs})
	b.WriteString(string(scanDirsYAML))
	b.WriteString("\n")

	// Entry directories
	if len(params.EntryDirs) > 0 {
		b.WriteString("# Additional entry directories (always included)\n")
		entryDirsYAML, _ := yaml.Marshal(map[string][]string{"entry_dirs": params.EntryDirs})
		b.WriteString(string(entryDirsYAML))
	} else {
		b.WriteString("# Additional entry directories (always included)\n")
		b.WriteString("# entry_dirs:\n")
		b.WriteString("#   - ~/special-project\n")
	}
	b.WriteString("\n")

	// Ignore directories
	b.WriteString("# Directory names to ignore when scanning\n")
	ignoreDirsYAML, _ := yaml.Marshal(map[string][]string{"ignore_dirs": params.IgnoreDirs})
	b.WriteString(string(ignoreDirsYAML))
	b.WriteString("\n")

	// Fallback session
	b.WriteString("# Fallback session for when killing the final session\n")
	fallbackYAML, _ := yaml.Marshal(map[string]models.Session{"fallback_session": params.FallbackSession})
	b.WriteString(string(fallbackYAML))
	b.WriteString("\n")

	// Tmux base
	b.WriteString("# Base index for tmux windows (0 or 1)\n")
	tmuxBaseYAML, _ := yaml.Marshal(map[string]int{"tmux_base": params.TmuxBase})
	b.WriteString(string(tmuxBaseYAML))
	b.WriteString("\n")

	// Default depth
	b.WriteString("# Default scanning depth for directories\n")
	defaultDepthYAML, _ := yaml.Marshal(map[string]int{"default_depth": params.DefaultDepth})
	b.WriteString(string(defaultDepthYAML))
	b.WriteString("\n")

	// Session layout
	b.WriteString("# Default layout for new tmux sessions\n")
	sessionLayoutYAML, _ := yaml.Marshal(map[string]models.SessionLayout{"session_layout": params.SessionLayout})
	b.WriteString(string(sessionLayoutYAML))
	b.WriteString("\n")

	// Editor
	b.WriteString("# Default editor editing this config file\n")
	editorYAML, _ := yaml.Marshal(map[string]string{"editor": params.Editor})
	b.WriteString(string(editorYAML))
	b.WriteString("\n")

	// Tmux Session Prefix
	b.WriteString("# The string that will prefix currently active Tmux sessions when using 'tms'\n")
	tmuxSessionPrefixYAML, _ := yaml.Marshal(map[string]string{"tmux_session_prefix": params.TmuxSessionPrefix})
	b.WriteString(string(tmuxSessionPrefixYAML))
	b.WriteString("\n")

	return b.String()
} // }}}
