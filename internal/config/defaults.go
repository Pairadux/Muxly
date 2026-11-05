// Package config holds muxly's config defaults, validation and template lookup.
package config

import "github.com/Pairadux/muxly/internal/models"

const (
	// DefaultEditor is the editor for "muxly config edit" when nothing else sets one.
	DefaultEditor = "vi"

	envVisual = "VISUAL"
	envEditor = "EDITOR"

	// DefaultTmuxSessionPrefix marks active sessions in the picker.
	DefaultTmuxSessionPrefix = "[TMUX] "
	// DefaultScanDepth is how deep scan_dirs are searched when no depth is set.
	DefaultScanDepth = 1
	// DefaultAlwaysKillOnLastSession keeps the prompt when killing the last session.
	DefaultAlwaysKillOnLastSession = false
)

var (
	// BaseIgnoreDirs are always filtered during scanning and cannot be overridden by user config.
	// There is no practical reason to scan inside these directories.
	BaseIgnoreDirs = []string{
		".git",
		"node_modules",
	}

	// DefaultScanDirs is the scan_dirs value in a new config.
	DefaultScanDirs = []models.ScanDir{}

	// DefaultEntryDirs is the entry_dirs value in a new config.
	DefaultEntryDirs = []models.EntryDir{
		{Path: "~"},
	}

	// DefaultIgnoreDirs is the ignore_dirs value in a new config.
	DefaultIgnoreDirs = []string{}

	// DefaultTemplates is the templates value in a new config.
	DefaultTemplates = []models.SessionTemplate{
		{
			Name:    "default",
			Label:   "Editor + Terminal",
			Default: true,
			Windows: []models.Window{
				{Name: "editor"},
				{Name: "term"},
			},
		},
		{
			Name:  "minimal",
			Label: "Single Window",
			Windows: []models.Window{
				{Name: "main"},
			},
		},
		{
			Name:  "quick",
			Label: "Quick Session",
			Path:  "~/",
			Windows: []models.Window{
				{Name: "main"},
			},
		},
	}
)

// NewDefaultConfig returns the config that "muxly config init" writes.
func NewDefaultConfig() models.Config {
	return models.Config{
		ScanDirs:   DefaultScanDirs,
		EntryDirs:  DefaultEntryDirs,
		IgnoreDirs: DefaultIgnoreDirs,
		Templates:  DefaultTemplates,
		Settings: models.Settings{
			DefaultDepth:            DefaultScanDepth,
			TmuxSessionPrefix:       DefaultTmuxSessionPrefix,
			AlwaysKillOnLastSession: DefaultAlwaysKillOnLastSession,
		},
	}
}

// ApplyDefaults fills in safe, mechanical settings defaults for fields
// the user is unlikely to have opinions about. Structural config like
// templates and directories are left alone — validation will catch
// those so the user can fix them intentionally (or use config init).
func ApplyDefaults(cfg *models.Config) {
	if cfg.Settings.TmuxSessionPrefix == "" {
		cfg.Settings.TmuxSessionPrefix = DefaultTmuxSessionPrefix
	}
	if cfg.Settings.DefaultDepth == 0 {
		cfg.Settings.DefaultDepth = DefaultScanDepth
	}
}
