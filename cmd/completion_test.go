//go:build unix

package cmd

import (
	"strings"
	"testing"
)

// The environment here has no config and nothing on PATH, like a fresh machine running
// eval "$(muxly completion zsh)" from a shell startup file.
func TestCompletionNeedsNoConfigOrTools(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			got := newCLI(t).run("completion", shell)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if !strings.Contains(got.stdout, "muxly") {
				t.Errorf("stdout doesn't look like a completion script:\n%.200s", got.stdout)
			}
		})
	}
}

// Shells run this hidden command on every Tab press once completion is installed.
func TestTabCompletionNeedsNoConfigOrTools(t *testing.T) {
	got := newCLI(t).run("__complete", "")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if !strings.Contains(got.stdout, "config") {
		t.Errorf("stdout doesn't list commands:\n%s", got.stdout)
	}
}

func TestOtherCommandsStillCheckConfigAndTools(t *testing.T) {
	got := newCLI(t).run("switch")

	if got.code == 0 || !strings.Contains(got.stderr, "missing required tools") {
		t.Errorf("exit %d, stderr %q", got.code, got.stderr)
	}
}
