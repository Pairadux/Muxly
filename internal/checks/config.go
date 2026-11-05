package checks

import (
	"errors"
	"fmt"
	"os"
	"strings"

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

// CheckConfig reports on the config as muxly loaded it, so that a config muxly would
// refuse always shows up here as an error.
func CheckConfig(path string, cfg *models.Config, report config.LoadReport) []CheckResult {
	results := []CheckResult{CheckConfigFile(path)}

	// If the file couldn't be read at all, checking what came of it would only add
	// complaints about everything being missing.
	readable := true
	var parseErr *config.ParseError
	switch {
	case errors.As(report.FileErr, &parseErr):
		results = append(results, configErrors(parseErr.Problems)...)
		results[len(results)-1].Hint = "Fix it with 'muxly config edit'"
		readable = parseErr.Partial
	case errors.Is(report.FileErr, config.ErrNotFound):
		// CheckConfigFile has covered it.
		readable = false
	case report.FileErr != nil:
		results = append(results, configErrors([]string{report.FileErr.Error()})...)
		readable = false
	}

	if report.EnvErr != nil {
		results = append(results, configErrors(strings.Split(report.EnvErr.Error(), "\n"))...)
	}

	for _, warning := range report.Warnings {
		results = append(results, CheckResult{Name: "config", Status: StatusWarning, Message: warning})
	}

	if readable {
		results = append(results, ValidateConfig(cfg)...)
	}
	return results
}

func configErrors(messages []string) []CheckResult {
	results := make([]CheckResult, len(messages))
	for i, message := range messages {
		results[i] = CheckResult{Name: "config", Status: StatusError, Message: message}
	}
	return results
}

// ValidateConfig reports everything config.Validate would refuse, which is what stops muxly
// running, or else a summary of what's configured.
func ValidateConfig(cfg *models.Config) []CheckResult {
	if problems := config.Problems(cfg); len(problems) > 0 {
		messages := make([]string, len(problems))
		for i, problem := range problems {
			messages[i] = problem.Error()
		}
		return configErrors(messages)
	}

	var detail string
	scanCount, entryCount := len(cfg.ScanDirs), len(cfg.EntryDirs)
	switch {
	case scanCount > 0 && entryCount > 0:
		detail = fmt.Sprintf("(%d scan, %d entry)", scanCount, entryCount)
	case scanCount > 0:
		detail = fmt.Sprintf("(%d scan)", scanCount)
	default:
		detail = fmt.Sprintf("(%d entry)", entryCount)
	}
	results := []CheckResult{{
		Name:    "directories",
		Status:  StatusOK,
		Message: "Directories configured",
		Detail:  detail,
	}}

	if dflt, found := config.DefaultTemplate(cfg); found {
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

	return results
}
