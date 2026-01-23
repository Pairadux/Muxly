package checks

import (
	"github.com/Pairadux/muxly/internal/models"
)

// ValidateConfig checks configuration for issues.
// This is designed for a future `muxly doctor` command.
func ValidateConfig(cfg *models.Config) []CheckResult {
	var results []CheckResult

	if len(cfg.ScanDirs) == 0 && len(cfg.EntryDirs) == 0 {
		results = append(results, CheckResult{
			Name:    "directories",
			Status:  StatusError,
			Message: "no directories configured (scan_dirs or entry_dirs required)",
		})
	} else {
		results = append(results, CheckResult{
			Name:    "directories",
			Status:  StatusOK,
			Message: "directories configured",
		})
	}

	if len(cfg.SessionLayout.Windows) == 0 {
		results = append(results, CheckResult{
			Name:    "session_layout",
			Status:  StatusError,
			Message: "session_layout must have at least one window",
		})
	} else {
		results = append(results, CheckResult{
			Name:    "session_layout",
			Status:  StatusOK,
			Message: "session layout is valid",
		})
	}

	if cfg.Settings.Editor == "" {
		results = append(results, CheckResult{
			Name:    "editor",
			Status:  StatusWarning,
			Message: "editor not set, will use default",
		})
	} else {
		results = append(results, CheckResult{
			Name:    "editor",
			Status:  StatusOK,
			Message: "editor configured: " + cfg.Settings.Editor,
		})
	}

	return results
}
