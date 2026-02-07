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

	if cfg.PrimaryTemplate.Name == "" {
		return fmt.Errorf("primary_template.name is required")
	}
	if len(cfg.PrimaryTemplate.Windows) == 0 {
		return fmt.Errorf("primary_template must have at least one window")
	}

	seenNames := map[string]bool{cfg.PrimaryTemplate.Name: true}
	for _, tmpl := range cfg.Templates {
		if tmpl.Name == "" {
			return fmt.Errorf("all templates must have a name")
		}
		if seenNames[tmpl.Name] {
			return fmt.Errorf("duplicate template name %q", tmpl.Name)
		}
		seenNames[tmpl.Name] = true
		if len(tmpl.Windows) == 0 {
			return fmt.Errorf("template %q must have at least one window", tmpl.Name)
		}
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
		if scanDir.Template != "" && !seenNames[scanDir.Template] {
			return fmt.Errorf("scan_dir %q references unknown template %q", scanDir.Path, scanDir.Template)
		}
	}

	for _, entryDir := range cfg.EntryDirs {
		if entryDir.Template != "" && !seenNames[entryDir.Template] {
			return fmt.Errorf("entry_dir %q references unknown template %q", entryDir.Path, entryDir.Template)
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
