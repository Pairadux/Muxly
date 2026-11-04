package config

import (
	"fmt"
	"strings"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/state"

	"github.com/spf13/cobra"
)

// listCmd lists the directories in the config
var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "List the directories in the config",
	Long: `List the scan_dirs, entry_dirs and ignore_dirs in the configuration file.

Paths are shown as written in the file. Each scan directory shows the depth it's
scanned to, including when that comes from default_depth.

Examples:
  muxly config list   # List all configured directories
  muxly config l      # Same, using the alias`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(formatDirectories(&state.Cfg))
		return nil
	},
}

func init() {
	Cmd.AddCommand(listCmd)
}

// formatDirectories renders the configured directories, one section per config key.
func formatDirectories(cfg *models.Config) string {
	scanDirs := make([]string, len(cfg.ScanDirs))
	for i, scanDir := range cfg.ScanDirs {
		details := []string{fmt.Sprintf("depth %d", scanDir.GetDepth(0, cfg.Settings.DefaultDepth))}
		if scanDir.Alias != "" {
			details = append(details, "alias "+scanDir.Alias)
		}
		if scanDir.Template != "" {
			details = append(details, "template "+scanDir.Template)
		}
		scanDirs[i] = fmt.Sprintf("%s (%s)", scanDir.Path, strings.Join(details, ", "))
	}

	entryDirs := make([]string, len(cfg.EntryDirs))
	for i, entryDir := range cfg.EntryDirs {
		entryDirs[i] = entryDir.Path
		if entryDir.Template != "" {
			entryDirs[i] = fmt.Sprintf("%s (template %s)", entryDir.Path, entryDir.Template)
		}
	}

	var sb strings.Builder
	writeSection(&sb, "scan_dirs", scanDirs)
	writeSection(&sb, "entry_dirs", entryDirs)
	writeSection(&sb, "ignore_dirs", cfg.IgnoreDirs)
	return sb.String()
}

func writeSection(sb *strings.Builder, title string, items []string) {
	fmt.Fprintf(sb, "%s:\n", title)
	if len(items) == 0 {
		sb.WriteString("  (none)\n")
		return
	}
	for _, item := range items {
		fmt.Fprintf(sb, "  %s\n", item)
	}
}
