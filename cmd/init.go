// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Pairadux/tms/internal/models"
	"github.com/spf13/cobra"
) // }}}

// Default values - defined once and used everywhere
var (
	defaultScanDirs = []models.ScanDir{
		{Path: "~/Dev", Depth: nil},
		{Path: "~/.dotfiles/dot_config/", Depth: nil},
	}
	defaultEntryDirs  = []string{"~/Documents", "~/Cloud"}
	defaultIgnoreDirs = []string{"~/Dev/_practice", "~/Dev/CS-Homework"}
	defaultTmuxBase   = 1
	defaultDepth      = 1
	defaultSession    = "Documents"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a new config file",
	Long: `Create a new config file

Creates a config file at the specified location (default location if no argument passed) if no config file exists.
Otherwise, the current config file is overwritten.
The flags provided are used to overwrite those values in the config file.
Any flags that are omitted will be assigned the default values shown.`,
	Run: func(cmd *cobra.Command, args []string) {
		scanDirs := getScanDirsOrDefault(cmd, "scan_dirs", defaultScanDirs)
		entryDirs := getStringArrayOrDefault(cmd, "entry_dirs", defaultEntryDirs)
		ignoreDirs := getStringArrayOrDefault(cmd, "ignore_dirs", defaultIgnoreDirs)
		tmuxBase := getIntOrDefault(cmd, "tmux_base", defaultTmuxBase)
		depth := getIntOrDefault(cmd, "default_depth", defaultDepth)
		session := getStringOrDefault(cmd, "default_session", defaultSession)
		sessionLayout := models.SessionLayout{
			Windows: []models.Window{
				{Name: "edit", Cmd: "nvim"},
				{Name: "term", Cmd: ""},
			},
		}

		configContent := generateConfigYAML(models.Config{
			ScanDirs:        scanDirs,
			EntryDirs:       entryDirs,
			IgnoreDirs:      ignoreDirs,
			FallbackSession: session,
			TmuxBase:        tmuxBase,
			DefaultDepth:    depth,
			SessionLayout:   sessionLayout,
		})

		// Handle the config content (write to file, etc.)
		fmt.Println(configContent)
	},
}

func init() { // {{{
	configCmd.AddCommand(initCmd)
	initCmd.Flags().IntP("tmux_base", "b", defaultTmuxBase, "What number your windows start ordering at.")
	initCmd.Flags().IntP("default_depth", "d", defaultDepth, "Default depth to scan.")
	initCmd.Flags().StringP("default_session", "D", defaultSession, "The name of the default session to fall back to.")
	initCmd.Flags().StringArrayP("scan_dirs", "s", scanDirsToStringArray(defaultScanDirs), "A list of paths that should always be scanned.\nConcat with :int for depth.")
	initCmd.Flags().StringArrayP("entry_dirs", "e", defaultEntryDirs, "A list of paths that are entries themselves.")
	initCmd.Flags().StringArrayP("ignore_dirs", "i", defaultIgnoreDirs, "A list of paths that should be removed.")
} // }}}

func generateConfigYAML(params models.Config) string {
	var b strings.Builder

	b.WriteString("# Configuration for Tmux Session Manager\n\n")

	b.WriteString("# Directories to scan for projects\n")
	b.WriteString("# Each entry can be a simple path or include depth:\n")
	b.WriteString("#   - path: ~/Dev\n")
	b.WriteString("#     depth: 3\n")
	b.WriteString("scan_dirs:\n")
	for _, dir := range params.ScanDirs {
		b.WriteString(fmt.Sprintf("  - path: %s\n", dir))
		if params.DefaultDepth > 0 {
			b.WriteString(fmt.Sprintf("    depth: %d\n", params.DefaultDepth))
		}
	}
	b.WriteString("\n")

	// Entry directories
	if len(params.EntryDirs) > 0 {
		b.WriteString("# Additional entry directories (always included)\n")
		b.WriteString("entry_dirs:\n")
		for _, dir := range params.EntryDirs {
			b.WriteString(fmt.Sprintf("  - %s\n", dir))
		}
	} else {
		b.WriteString("# Additional entry directories (always included)\n")
		b.WriteString("# entry_dirs:\n")
		b.WriteString("#   - ~/special-project\n")
	}
	b.WriteString("\n")

	// Ignore directories
	b.WriteString("# Directory names to ignore when scanning\n")
	b.WriteString("ignore_dirs:\n")
	for _, dir := range params.IgnoreDirs {
		b.WriteString(fmt.Sprintf("  - %s\n", dir))
	}
	b.WriteString("\n")

	// Fallback session
	b.WriteString("# Default session name when no project is selected\n")
	b.WriteString(fmt.Sprintf("fallback_session: %s\n\n", params.FallbackSession))

	// Tmux base
	b.WriteString("# Base index for tmux windows (0 or 1)\n")
	b.WriteString(fmt.Sprintf("tmux_base: %d\n\n", params.TmuxBase))

	// Default depth
	b.WriteString("# Default scanning depth for directories\n")
	b.WriteString(fmt.Sprintf("default_depth: %d\n\n", params.DefaultDepth))

	// Session layout
	b.WriteString("# Default layout for new tmux sessions\n")
	b.WriteString("session_layout:\n")
	b.WriteString("  windows:\n")
	b.WriteString("    - name: edit\n")
	b.WriteString("      cmd: nvim\n")
	b.WriteString("    - name: term\n")
	b.WriteString("      # cmd: (empty for default shell)\n")

	return b.String()
}

// Helper functions to get flag values with defaults
func getStringArrayOrDefault(cmd *cobra.Command, flag string, defaultVal []string) []string {
	if val, err := cmd.Flags().GetStringArray(flag); err == nil && len(val) > 0 {
		return val
	}
	return defaultVal
}

func getStringOrDefault(cmd *cobra.Command, flag string, defaultVal string) string {
	if val, err := cmd.Flags().GetString(flag); err == nil && val != "" {
		return val
	}
	return defaultVal
}

func getIntOrDefault(cmd *cobra.Command, flag string, defaultVal int) int {
	if val, err := cmd.Flags().GetInt(flag); err == nil && cmd.Flags().Changed(flag) {
		return val
	}
	return defaultVal
}

// Helper functions to get flag values with defaults
func getScanDirsOrDefault(cmd *cobra.Command, flag string, defaultVal []models.ScanDir) []models.ScanDir {
	if val, err := cmd.Flags().GetStringArray(flag); err == nil && len(val) > 0 {
		return parseScanDirs(val)
	}
	return defaultVal
}

// Helper function to convert ScanDir slice to string slice for flag defaults
func scanDirsToStringArray(scanDirs []models.ScanDir) []string {
	result := make([]string, len(scanDirs))
	for i, sd := range scanDirs {
		if sd.Depth != nil {
			result[i] = fmt.Sprintf("%s:%d", sd.Path, *sd.Depth)
		} else {
			result[i] = sd.Path
		}
	}
	return result
}

// Helper function to parse string array back to ScanDir slice
func parseScanDirs(paths []string) []models.ScanDir {
	result := make([]models.ScanDir, len(paths))
	for i, path := range paths {
		parts := strings.Split(path, ":")
		scanDir := models.ScanDir{Path: parts[0]}

		if len(parts) > 1 {
			if depth, err := strconv.Atoi(parts[1]); err == nil {
				scanDir.Depth = &depth
			}
		}

		result[i] = scanDir
	}
	return result
}
