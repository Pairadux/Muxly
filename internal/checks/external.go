package checks

import (
	"fmt"
	"os/exec"
	"strings"
)

// VerifyExternalUtils checks for required external tools (tmux, fzf).
// Returns an error if any required tools are missing.
func VerifyExternalUtils() error {
	var missing []string

	if _, err := exec.LookPath("tmux"); err != nil {
		missing = append(missing, "tmux")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		missing = append(missing, "fzf")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}

	return nil
}

// CheckExternalUtils returns detailed check results for external tools.
// This is designed for a future `muxly doctor` command.
func CheckExternalUtils() []CheckResult {
	var results []CheckResult

	if _, err := exec.LookPath("tmux"); err != nil {
		results = append(results, CheckResult{
			Name:    "tmux",
			Status:  StatusError,
			Message: "tmux not found in PATH",
		})
	} else {
		results = append(results, CheckResult{
			Name:    "tmux",
			Status:  StatusOK,
			Message: "tmux is installed",
		})
	}

	if _, err := exec.LookPath("fzf"); err != nil {
		results = append(results, CheckResult{
			Name:    "fzf",
			Status:  StatusError,
			Message: "fzf not found in PATH",
		})
	} else {
		results = append(results, CheckResult{
			Name:    "fzf",
			Status:  StatusOK,
			Message: "fzf is installed",
		})
	}

	return results
}
