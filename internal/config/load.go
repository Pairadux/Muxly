package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/models"
)

const (
	envVisual = "VISUAL"
	envEditor = "EDITOR"

	envMuxlyEditor   = "MUXLY_EDITOR"
	envDefaultDepth  = "MUXLY_DEFAULT_DEPTH"
	envSessionPrefix = "MUXLY_TMUX_SESSION_PREFIX"
	envAlwaysKill    = "MUXLY_ALWAYS_KILL_ON_LAST_SESSION"
)

// ErrNotFound reports that there's no config file where muxly looked.
var ErrNotFound = errors.New("no config file found")

// LoadReport says what went wrong loading the config. Commands that fix a broken config,
// or report on one, carry on regardless. Everything else stops at Err.
type LoadReport struct {
	// FileErr is what's wrong with the config file: ErrNotFound, a *ParseError, or
	// a failure to read it.
	FileErr error
	// EnvErr is what's wrong with the MUXLY_ environment variables.
	EnvErr error
	// Warnings are about things that still work but should be changed.
	Warnings []string
}

// Err returns everything that stops the config being used, or nil.
func (r LoadReport) Err() error {
	return errors.Join(r.FileErr, r.EnvErr)
}

// DefaultPath returns where the config file is unless --config says otherwise.
func DefaultPath() (string, error) {
	dir := os.Getenv(constants.EnvXdgConfigHome)
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("finding the config directory: %w", err)
		}
	}
	return filepath.Join(dir, "muxly", "config.yaml"), nil
}

// Load reads the config file at path, then applies environment variables and defaults on
// top. It always returns a config: as much as could be worked out, which is what lets
// "muxly config edit" still find an editor when the file is broken. The report says how
// far to trust it.
func Load(path string) (models.Config, LoadReport) {
	var cfg models.Config
	var report LoadReport

	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		report.FileErr = notFoundError(path)
	case err != nil:
		report.FileErr = fmt.Errorf("reading config: %w", err)
	default:
		var warnings []string
		cfg, warnings, err = Parse(data)
		for _, warning := range warnings {
			report.Warnings = append(report.Warnings, path+": "+warning)
		}
		if err != nil {
			report.FileErr = InvalidFileError(path, err)
		}
	}

	report.EnvErr = applyEnv(&cfg)
	ApplyDefaults(&cfg)
	return cfg, report
}

func notFoundError(path string) error {
	return fmt.Errorf("%w at %s\nRun 'muxly config init' to create one, or use --config to specify a path", ErrNotFound, path)
}

// InvalidFileError says that the config file at path has the given problems, and what to
// do about them.
func InvalidFileError(path string, problems error) error {
	return fmt.Errorf("invalid config file %s\n%w\nFix it with 'muxly config edit'", path, problems)
}

// applyEnv overlays the MUXLY_ environment variables on the config. An empty one counts as
// not set, and one that can't be used is an error rather than quietly doing nothing.
func applyEnv(cfg *models.Config) error {
	var problems []error

	if editor := os.Getenv(envMuxlyEditor); strings.TrimSpace(editor) != "" {
		cfg.Settings.Editor = editor
	}

	if prefix := os.Getenv(envSessionPrefix); prefix != "" {
		cfg.Settings.TmuxSessionPrefix = prefix
	}

	if value := os.Getenv(envDefaultDepth); value != "" {
		depth, err := strconv.Atoi(value)
		if err != nil || depth < 1 {
			problems = append(problems, fmt.Errorf("%s must be a whole number of at least 1, got %q", envDefaultDepth, value))
		} else {
			cfg.Settings.DefaultDepth = depth
		}
	}

	if value := os.Getenv(envAlwaysKill); value != "" {
		alwaysKill, err := strconv.ParseBool(value)
		if err != nil {
			problems = append(problems, fmt.Errorf("%s must be true or false, got %q", envAlwaysKill, value))
		} else {
			cfg.Settings.AlwaysKillOnLastSession = alwaysKill
		}
	}

	return errors.Join(problems...)
}
