package config

import (
	"fmt"
	"os"

	"github.com/Pairadux/muxly/internal/models"
	"gopkg.in/yaml.v3"
)

// Validate ensures that the application configuration is valid and complete.
// It checks that at least one directory is configured for scanning and that
// the session layout has at least one window.
func Validate(cfg *models.Config) error {
	if len(cfg.ScanDirs) == 0 && len(cfg.EntryDirs) == 0 {
		return fmt.Errorf("no directories configured for scanning (scan_dirs or entry_dirs required)")
	}

	if len(cfg.SessionLayout.Windows) == 0 {
		return fmt.Errorf("session_layout must have at least one window")
	}

	seenAliases := make(map[string]string)
	for _, scanDir := range cfg.ScanDirs {
		if scanDir.Alias != "" {
			if existingPath, exists := seenAliases[scanDir.Alias]; exists {
				return fmt.Errorf("duplicate alias %q used by both %q and %q",
					scanDir.Alias, existingPath, scanDir.Path)
			}
			seenAliases[scanDir.Alias] = scanDir.Path
		}
	}

	return nil
}

// ValidateConfigFile reads and validates a config file at the given path.
// Returns the parsed config if valid, or an error if the file cannot be read or is invalid.
func ValidateConfigFile(path string) (*models.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	var cfg models.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("invalid YAML syntax: %w", err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}
