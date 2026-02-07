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
	DefaultTmuxBase                = 1
	DefaultScanDepth               = 1
	DefaultAlwaysKillOnLastSession = false
)

var (
	DefaultScanDirs = []models.ScanDir{}

	DefaultEntryDirs = []models.EntryDir{
		{Path: "~"},
	}

	DefaultIgnoreDirs = []string{
		".git",
		"node_modules",
	}

	DefaultPrimaryTemplate = models.SessionTemplate{
		Name: "Editor + Terminal",
		Windows: []models.Window{
			{Name: "editor"},
			{Name: "term"},
		},
	}

	DefaultTemplates = []models.SessionTemplate{
		{
			Name: "Single Window",
			Windows: []models.Window{
				{Name: "main"},
			},
		},
		{
			Name: "Quick Session",
			Path: "~/",
			Windows: []models.Window{
				{Name: "main"},
			},
		},
	}
)

func NewDefaultConfig() models.Config {
	return models.Config{
		ScanDirs:        DefaultScanDirs,
		EntryDirs:       DefaultEntryDirs,
		IgnoreDirs:      DefaultIgnoreDirs,
		PrimaryTemplate: DefaultPrimaryTemplate,
		Templates:       DefaultTemplates,
		Settings: models.Settings{
			Editor:                  DefaultEditor,
			TmuxBase:                DefaultTmuxBase,
			DefaultDepth:            DefaultScanDepth,
			TmuxSessionPrefix:       DefaultTmuxSessionPrefix,
			AlwaysKillOnLastSession: DefaultAlwaysKillOnLastSession,
		},
	}
}

func ApplyDefaults(cfg *models.Config) {
	if cfg.Settings.Editor == "" {
		cfg.Settings.Editor = DefaultEditor
	}
	if cfg.PrimaryTemplate.Name == "" {
		cfg.PrimaryTemplate = DefaultPrimaryTemplate
	}
	if cfg.Settings.TmuxSessionPrefix == "" {
		cfg.Settings.TmuxSessionPrefix = DefaultTmuxSessionPrefix
	}
	if cfg.Settings.TmuxBase == 0 {
		cfg.Settings.TmuxBase = DefaultTmuxBase
	}
	if cfg.Settings.DefaultDepth == 0 {
		cfg.Settings.DefaultDepth = DefaultScanDepth
	}
}
