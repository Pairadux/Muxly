package checks

import (
	"fmt"
	"os"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
)

// CheckConfigFile validates that the config file exists and is readable.
func CheckConfigFile(path string) CheckResult {
	if path == "" {
		return CheckResult{
			Name:    "config_file",
			Status:  StatusError,
			Message: "No config file path specified",
			Hint:    "Run 'muxly config init' to create a config file",
		}
	}

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return CheckResult{
			Name:    "config_file",
			Status:  StatusError,
			Message: "Config file not found",
			Detail:  fmt.Sprintf("(%s)", path),
			Hint:    "Run 'muxly config init' to create a config file",
		}
	}
	if err != nil {
		return CheckResult{
			Name:    "config_file",
			Status:  StatusError,
			Message: "Cannot access config file",
			Detail:  fmt.Sprintf("(%s)", path),
			Hint:    fmt.Sprintf("Error: %v", err),
		}
	}
	if info.IsDir() {
		return CheckResult{
			Name:    "config_file",
			Status:  StatusError,
			Message: "Config path is a directory, not a file",
			Detail:  fmt.Sprintf("(%s)", path),
		}
	}

	return CheckResult{
		Name:    "config_file",
		Status:  StatusOK,
		Message: "Config file",
		Detail:  fmt.Sprintf("(%s)", path),
	}
}

// ValidateConfig checks configuration values for issues.
func ValidateConfig(cfg *models.Config) []CheckResult {
	var results []CheckResult

	scanCount := len(cfg.ScanDirs)
	entryCount := len(cfg.EntryDirs)
	if scanCount == 0 && entryCount == 0 {
		results = append(results, CheckResult{
			Name:    "directories",
			Status:  StatusError,
			Message: "No directories configured",
			Hint:    "Add paths to scan_dirs or entry_dirs in config",
		})
	} else {
		detail := ""
		if scanCount > 0 && entryCount > 0 {
			detail = fmt.Sprintf("(%d scan, %d entry)", scanCount, entryCount)
		} else if scanCount > 0 {
			detail = fmt.Sprintf("(%d scan)", scanCount)
		} else {
			detail = fmt.Sprintf("(%d entry)", entryCount)
		}
		results = append(results, CheckResult{
			Name:    "directories",
			Status:  StatusOK,
			Message: "Directories configured",
			Detail:  detail,
		})
	}

	dflt, hasDefault := config.DefaultTemplate(cfg)
	if !hasDefault {
		results = append(results, CheckResult{
			Name:    "default_template",
			Status:  StatusError,
			Message: "No default template set",
			Hint:    "Set default: true on one template in the templates list",
		})
	} else if len(dflt.Windows) == 0 {
		results = append(results, CheckResult{
			Name:    "default_template",
			Status:  StatusError,
			Message: "Default template has no windows",
			Hint:    "Add at least one window to the default template",
		})
	} else {
		results = append(results, CheckResult{
			Name:    "default_template",
			Status:  StatusOK,
			Message: "Default template",
			Detail:  fmt.Sprintf("(%d window(s))", len(dflt.Windows)),
		})
	}

	if len(cfg.Templates) > 1 {
		results = append(results, CheckResult{
			Name:    "templates",
			Status:  StatusOK,
			Message: "Additional templates",
			Detail:  fmt.Sprintf("(%d)", len(cfg.Templates)-1),
		})
	}

	if cfg.Settings.TmuxBase != 0 && cfg.Settings.TmuxBase != 1 {
		results = append(results, CheckResult{
			Name:    "tmux_base",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Unusual tmux_base value: %d", cfg.Settings.TmuxBase),
			Hint:    "tmux_base is typically 0 or 1",
		})
	}

	return results
}
