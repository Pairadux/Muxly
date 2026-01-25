package checks

import (
	"fmt"
	"os"
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

func getVersion(cmd string, args ...string) string {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return ""
	}
	version := strings.TrimSpace(string(out))
	version = strings.TrimPrefix(version, "tmux ")
	if idx := strings.Index(version, " "); idx != -1 {
		version = version[:idx]
	}
	return version
}

func checkTool(name, versionFlag, missingHint string) CheckResult {
	if _, err := exec.LookPath(name); err != nil {
		return CheckResult{
			Name:    name,
			Status:  StatusError,
			Message: fmt.Sprintf("%s not found in PATH", name),
			Hint:    missingHint,
		}
	}

	version := getVersion(name, versionFlag)
	detail := ""
	if version != "" {
		detail = fmt.Sprintf("(%s)", version)
	}

	return CheckResult{
		Name:    name,
		Status:  StatusOK,
		Message: name,
		Detail:  detail,
	}
}

// CheckExternalUtils returns detailed check results for external tools.
func CheckExternalUtils() []CheckResult {
	return []CheckResult{
		checkTool("tmux", "-V", "Install tmux: https://github.com/tmux/tmux"),
		checkTool("fzf", "--version", "Install fzf: https://github.com/junegunn/fzf"),
	}
}

// CheckEditor validates the configured editor or falls back to environment.
func CheckEditor(configEditor string) CheckResult {
	editor := configEditor
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}

	if editor == "" {
		return CheckResult{
			Name:    "editor",
			Status:  StatusWarning,
			Message: "No editor configured",
			Hint:    "Set settings.editor in config or $EDITOR environment variable",
		}
	}

	cmd := strings.Fields(editor)[0]
	if _, err := exec.LookPath(cmd); err != nil {
		return CheckResult{
			Name:    "editor",
			Status:  StatusWarning,
			Message: fmt.Sprintf("Editor '%s' not found in PATH", cmd),
			Hint:    "Verify the editor command is installed and in your PATH",
		}
	}

	return CheckResult{
		Name:    "editor",
		Status:  StatusOK,
		Message: "Editor",
		Detail:  fmt.Sprintf("(%s)", editor),
	}
}
