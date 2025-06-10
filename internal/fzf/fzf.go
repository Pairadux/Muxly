// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package fzf

// IMPORTS {{{
import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)// }}}

func SelectWithFzf(options []string) (string, error) {
	fzf := exec.Command("fzf")
	fzf.Stdin = strings.NewReader(strings.Join(options, "\n"))
	fzf.Stderr = os.Stderr
	choice, err := fzf.Output()
	if err != nil {
		// Exit gracefully if user quits
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 130 {
			return "", fmt.Errorf("user cancelled")
		}
		return "", err
	}
	return strings.TrimSpace(string(choice)), nil
}
