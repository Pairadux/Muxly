package config

import "github.com/Pairadux/muxly/internal/models"

// TODO: add config option for "use-absolute-path"
// This would change the entries from using the basename to using the resolved absolute path in the fzf selector
// TODO: add config option for "use-home-based-path"
// Similar to use-absolute-path but shows paths from ~/ rather than /
// Would need to prioritize one over the other if both are enabled and detail which takes priority
// TODO: add a config option to remove current session from list of options
// Might would help with the duplicate problem, especially in conjuction with absolute path config option

// Simple default values (constants)
const (
	DefaultEditor                  = "vi"
	DefaultFallbackSessionName     = "Default"
	DefaultFallbackSessionPath     = "~/"
	DefaultTmuxSessionPrefix       = "[TMUX] "
	DefaultTmuxBase                = 1
	DefaultScanDepth               = 1
	DefaultAlwaysKillOnLastSession = false
)

// Complex default values (variables)
// These are intentionally minimal - just enough for the program to run.
// Users should customize these in their config file.
var (
	DefaultScanDirs = []models.ScanDir{}

	DefaultEntryDirs = []string{
		"~", // Home directory - universally available
	}

	DefaultIgnoreDirs = []string{}

	DefaultSessionLayout = models.SessionLayout{
		Windows: []models.Window{
			{Name: "main", Cmd: ""}, // Single window with default shell
		},
	}

	DefaultFallbackSession = models.Session{
		Name:   DefaultFallbackSessionName,
		Path:   DefaultFallbackSessionPath,
		Layout: DefaultSessionLayout, // Reuse the default session layout
	}
)

// NewDefaultConfig returns a new Config struct with all default values
func NewDefaultConfig() models.Config {
	return models.Config{
		ScanDirs:                DefaultScanDirs,
		EntryDirs:               DefaultEntryDirs,
		IgnoreDirs:              DefaultIgnoreDirs,
		FallbackSession:         DefaultFallbackSession,
		TmuxBase:                DefaultTmuxBase,
		DefaultDepth:            DefaultScanDepth,
		SessionLayout:           DefaultSessionLayout,
		Editor:                  DefaultEditor,
		TmuxSessionPrefix:       DefaultTmuxSessionPrefix,
		AlwaysKillOnLastSession: DefaultAlwaysKillOnLastSession,
	}
}

// ApplyDefaults fills in any missing values in the provided config with defaults
func ApplyDefaults(cfg *models.Config) {
	if cfg.Editor == "" {
		cfg.Editor = DefaultEditor
	}
	if cfg.FallbackSession.Name == "" {
		cfg.FallbackSession.Name = DefaultFallbackSessionName
	}
	if cfg.FallbackSession.Path == "" {
		cfg.FallbackSession.Path = DefaultFallbackSessionPath
	}
	if len(cfg.FallbackSession.Layout.Windows) == 0 {
		cfg.FallbackSession.Layout = cfg.SessionLayout
	}
	if cfg.TmuxSessionPrefix == "" {
		cfg.TmuxSessionPrefix = DefaultTmuxSessionPrefix
	}
	if cfg.TmuxBase == 0 {
		cfg.TmuxBase = DefaultTmuxBase
	}
	if cfg.DefaultDepth == 0 {
		cfg.DefaultDepth = DefaultScanDepth
	}
}
