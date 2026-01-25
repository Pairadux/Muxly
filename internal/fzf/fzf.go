package fzf


import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Pairadux/muxly/internal/constants"
)

// SelectWithFzf presents a list of options to the user via the fzf fuzzy finder
// and returns the selected option. The options are passed as stdin to the fzf
// command, allowing the user to interactively filter and select from them.
//
// Returns an error if fzf is not available, if there's an I/O error, or if the
// user cancels the selection (Ctrl+C), in which case the error message is
// "user cancelled".
func SelectWithFzf(options []string) (string, error) {
	fzf := exec.Command("fzf")
	fzf.Stdin = strings.NewReader(strings.Join(options, "\n"))
	fzf.Stderr = os.Stderr
	choice, err := fzf.Output()
	if err != nil {
		// Exit gracefully if user quits
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == constants.FzfUserCancelExitCode {
			return "", fmt.Errorf(constants.UserCancelledMsg)
		}
		return "", err
	}
	return strings.TrimSpace(string(choice)), nil
}
