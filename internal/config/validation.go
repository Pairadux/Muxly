package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/Pairadux/muxly/internal/models"
)

// Validate returns everything wrong with the config as one error, or nil. See Problems.
func Validate(cfg *models.Config) error {
	return errors.Join(Problems(cfg)...)
}

// Problems returns everything that makes the config unusable. It checks that at least one
// directory is configured, and that exactly one template is marked as default, with every
// template having a name and at least one window.
func Problems(cfg *models.Config) []error {
	var problems []error

	if len(cfg.ScanDirs) == 0 && len(cfg.EntryDirs) == 0 {
		problems = append(problems, errors.New("no directories configured for scanning (scan_dirs or entry_dirs required)"))
	}

	seenNames := make(map[string]bool)
	if len(cfg.Templates) == 0 {
		problems = append(problems, errors.New("at least one template is required"))
	} else {
		defaultCount := 0
		for _, tmpl := range cfg.Templates {
			if tmpl.Name == "" {
				problems = append(problems, errors.New("all templates must have a name"))
			} else if seenNames[tmpl.Name] {
				problems = append(problems, fmt.Errorf("duplicate template name %q", tmpl.Name))
			}
			seenNames[tmpl.Name] = true
			if len(tmpl.Windows) == 0 {
				problems = append(problems, fmt.Errorf("template %q must have at least one window", tmpl.Name))
			}
			if tmpl.Default {
				defaultCount++
			}
		}

		if defaultCount == 0 {
			problems = append(problems, errors.New("exactly one template must have default: true"))
		}
		if defaultCount > 1 {
			problems = append(problems, fmt.Errorf("only one template can have default: true, found %d", defaultCount))
		}
	}

	seenAliases := make(map[string]string)
	for _, scanDir := range cfg.ScanDirs {
		if scanDir.Depth != nil && *scanDir.Depth < 1 {
			problems = append(problems, fmt.Errorf("scan_dir %q has depth %d, and it must be at least 1", scanDir.Path, *scanDir.Depth))
		}
		if scanDir.Alias != "" {
			if existingPath, exists := seenAliases[scanDir.Alias]; exists {
				problems = append(problems, fmt.Errorf("duplicate alias %q used by both %q and %q", scanDir.Alias, existingPath, scanDir.Path))
			}
			seenAliases[scanDir.Alias] = scanDir.Path
		}
		if scanDir.Template != "" && !seenNames[scanDir.Template] {
			problems = append(problems, fmt.Errorf("scan_dir %q references unknown template %q", scanDir.Path, scanDir.Template))
		}
	}

	for _, entryDir := range cfg.EntryDirs {
		if entryDir.Template != "" && !seenNames[entryDir.Template] {
			problems = append(problems, fmt.Errorf("entry_dir %q references unknown template %q", entryDir.Path, entryDir.Template))
		}
	}

	return problems
}

// ValidateConfigFile checks a config file on its own, without the environment or defaults,
// as "muxly config edit" does after the editor closes. It returns the file's warnings, and
// an error if it can't be read or muxly wouldn't accept it.
func ValidateConfigFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	cfg, warnings, err := Parse(data)
	if err != nil {
		return warnings, err
	}

	if err := Validate(&cfg); err != nil {
		return warnings, fmt.Errorf("invalid config: %w", err)
	}

	return warnings, nil
}
