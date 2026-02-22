package config

import "github.com/Pairadux/muxly/internal/models"

// TODO: add config option for "use-absolute-path"
// This would change the entries from using the basename to using the resolved absolute path in the fzf selector
// TODO: add config option for "use-home-based-path"
// Similar to use-absolute-path but shows paths from ~/ rather than /
// Would need to prioritize one over the other if both are enabled and detail which takes priority
// TODO: add a config option to remove current session from list of options
// Might would help with the duplicate problem, especially in conjuction with absolute path config option

const (
	DefaultEditor                  = "vi"
	DefaultTmuxSessionPrefix       = "[TMUX] "
	DefaultTmuxBase                = 0
	DefaultScanDepth               = 1
	DefaultAlwaysKillOnLastSession = false
)

var (
	// BaseIgnoreDirs are always filtered during scanning and cannot be overridden by user config.
	// There is no practical reason to scan inside these directories.
	BaseIgnoreDirs = []string{
		".git",
		"node_modules",
	}

	DefaultScanDirs = []models.ScanDir{}

	DefaultEntryDirs = []models.EntryDir{
		{Path: "~"},
	}

	DefaultIgnoreDirs = []string{}

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

func NewDefaultConfig() models.Config {
	return models.Config{
		ScanDirs:   DefaultScanDirs,
		EntryDirs:  DefaultEntryDirs,
		IgnoreDirs: DefaultIgnoreDirs,
		Templates:  DefaultTemplates,
		Settings: models.Settings{
			Editor:                  DefaultEditor,
			TmuxBase:                DefaultTmuxBase,
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
	if cfg.Settings.Editor == "" {
		cfg.Settings.Editor = DefaultEditor
	}
	if cfg.Settings.TmuxSessionPrefix == "" {
		cfg.Settings.TmuxSessionPrefix = DefaultTmuxSessionPrefix
	}
	if cfg.Settings.TmuxBase < 0 {
		cfg.Settings.TmuxBase = DefaultTmuxBase
	}
	if cfg.Settings.DefaultDepth == 0 {
		cfg.Settings.DefaultDepth = DefaultScanDepth
	}
}
