// Package fzf runs the fzf picker.
package fzf

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/Pairadux/muxly/internal/constants"
)

// ErrCancelled reports that the picker closed without a choice: the user pressed Esc or
// Ctrl-C, or pressed Enter with nothing matching.
// Callers should treat it as a normal exit, not a failure.
var ErrCancelled = errors.New("selection cancelled")

// SelectWithFzf presents a list of options to the user via the fzf fuzzy finder
// and returns the selected option. The options are passed as stdin to the fzf
// command, allowing the user to interactively filter and select from them.
//
// Returns ErrCancelled if the picker closes without a choice, or another error if
// fzf is not available or fails.
func SelectWithFzf(options []string) (string, error) {
	fzf := exec.Command("fzf")
	fzf.Stdin = strings.NewReader(strings.Join(options, "\n"))
	fzf.Stderr = os.Stderr
	choice, err := fzf.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			switch exitErr.ExitCode() {
			case constants.FzfNoMatchExitCode, constants.FzfUserCancelExitCode:
				return "", ErrCancelled
			}
		}
		return "", err
	}
	return strings.TrimSpace(string(choice)), nil
}
