// Package fzf runs the fzf picker.
package fzf

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"

	"github.com/Pairadux/muxly/internal/constants"
)

// ErrCancelled reports that the picker closed without a choice: the user pressed Esc or
// Ctrl-C, or pressed Enter with nothing matching.
// Callers should treat it as a normal exit, not a failure.
var ErrCancelled = errors.New("selection cancelled")

// Pick shows labels in the fzf fuzzy finder and returns the index of the chosen one.
//
// Each line given to fzf is the label's index, a tab, then the label. fzf shows and
// searches only the label but returns the whole line, so the choice never has to be
// worked out from what's displayed, and two rows with the same label stay different rows.
//
// Returns ErrCancelled if the picker closes without a choice, or another error if fzf is
// not available or fails.
func Pick(labels []string) (int, error) {
	var input strings.Builder
	for i, label := range labels {
		fmt.Fprintf(&input, "%d\t%s\n", i, printable(label))
	}

	fzf := exec.Command("fzf", "--delimiter=\t", "--with-nth=2..")
	fzf.Stdin = strings.NewReader(input.String())
	fzf.Stderr = os.Stderr
	choice, err := fzf.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			switch exitErr.ExitCode() {
			case constants.FzfNoMatchExitCode, constants.FzfUserCancelExitCode:
				return 0, ErrCancelled
			}
		}
		return 0, err
	}

	key, _, _ := strings.Cut(string(choice), "\t")
	i, err := strconv.Atoi(key)
	if err != nil || i < 0 || i >= len(labels) {
		return 0, fmt.Errorf("unexpected output from fzf: %q", choice)
	}
	return i, nil
}

// SelectWithFzf shows options in the picker and returns the chosen one. See Pick for the
// errors it returns.
func SelectWithFzf(options []string) (string, error) {
	i, err := Pick(options)
	if err != nil {
		return "", err
	}
	return options[i], nil
}

// printable replaces control characters, because a directory's name can hold a tab or
// newline, and either would break the one-line-per-row format fzf reads.
func printable(label string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return '?'
		}
		return r
	}, label)
}
