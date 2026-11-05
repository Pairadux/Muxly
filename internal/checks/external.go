package checks

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// minTmuxVersion is the oldest tmux that muxly supports. CI runs the tests on it, in the
// oldest-tmux job of .github/workflows/test.yml, so change the two together.
const minTmuxVersion = "3.2"

// tmuxVersionPattern finds the number in a tmux version, such as "3.2a" or "next-3.7".
var tmuxVersionPattern = regexp.MustCompile(`^(?:next-)?(\d+)\.(\d+)`)

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

// parseTmuxVersion returns the major and minor numbers of a tmux version.
func parseTmuxVersion(version string) ([]int, bool) {
	match := tmuxVersionPattern.FindStringSubmatch(version)
	if match == nil {
		return nil, false
	}
	major, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, false
	}
	minor, err := strconv.Atoi(match[2])
	if err != nil {
		return nil, false
	}
	return []int{major, minor}, true
}

// tmuxBelowMinimum reports whether a tmux version is older than muxly supports. A build
// that doesn't state a number, such as "master" or "openbsd-7.4", gets the benefit of the
// doubt. The letter in a version like "3.2a" marks a bug-fix release, so it's ignored.
func tmuxBelowMinimum(version string) bool {
	parsed, ok := parseTmuxVersion(version)
	if !ok {
		return false
	}
	minimum, ok := parseTmuxVersion(minTmuxVersion)
	if !ok {
		return false
	}
	return slices.Compare(parsed, minimum) < 0
}

// checkTmux checks that tmux is installed and new enough. An older tmux only gets a
// warning, because muxly may still work there, just untested.
func checkTmux() CheckResult {
	result := checkTool("tmux", "-V", "Install tmux: https://github.com/tmux/tmux")
	if result.Status != StatusOK {
		return result
	}

	if version := getVersion("tmux", "-V"); tmuxBelowMinimum(version) {
		result.Status = StatusWarning
		result.Message = fmt.Sprintf("tmux %s is older than %s, the oldest version muxly supports", version, minTmuxVersion)
		result.Detail = ""
		result.Hint = "Upgrade tmux: https://github.com/tmux/tmux"
	}
	return result
}

// CheckExternalUtils returns detailed check results for external tools.
func CheckExternalUtils() []CheckResult {
	return []CheckResult{
		checkTmux(),
		checkTool("fzf", "--version", "Install fzf: https://github.com/junegunn/fzf"),
	}
}

// firstNonBlank returns the first value that has something besides whitespace, or "".
func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// CheckEditor validates the configured editor or falls back to environment.
func CheckEditor(configEditor string) CheckResult {
	editor := firstNonBlank(configEditor, os.Getenv("EDITOR"), os.Getenv("VISUAL"))
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
