//go:build unix

package cmd

import (
	"strings"
	"testing"
)

func TestDoctorSurvivesBlankEditor(t *testing.T) {
	for _, editor := range []string{" ", "\t "} {
		c := newCLI(t).withFakes().writeConfig(validConfig).setEnv("EDITOR=" + editor)

		got := c.run("doctor")

		if strings.Contains(got.stderr, "panic") {
			t.Fatalf("doctor panicked:\n%s", got.stderr)
		}
		if got.code != 0 {
			t.Errorf("exit %d, stdout %q, stderr %q", got.code, got.stdout, got.stderr)
		}
		if !strings.Contains(got.stdout, "No editor configured") {
			t.Errorf("stdout doesn't mention the missing editor:\n%s", got.stdout)
		}
	}
}

func TestDoctorWarnsBelowTheMinimumTmux(t *testing.T) {
	tests := []struct {
		version string
		warns   bool
	}{
		{version: "3.1c", warns: true},
		{version: "3.2a"},
		{version: "3.8"},
		{version: "master"},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig).setEnv("FAKE_TMUX_VERSION=" + tt.version)

			got := c.run("doctor")

			// It's a warning, so doctor still passes.
			if got.code != 0 {
				t.Errorf("exit %d, stdout %q", got.code, got.stdout)
			}
			if warned := strings.Contains(got.stdout, "the oldest version muxly supports"); warned != tt.warns {
				t.Errorf("warned = %v, want %v:\n%s", warned, tt.warns, got.stdout)
			}
		})
	}
}

// Doctor's version check must run on the real tmux, whichever version that is.
func TestDoctorReadsTheRealTmuxVersion(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	c.withRealTmux()

	got := c.run("doctor")

	if !strings.Contains(got.stdout, "tmux") || strings.Contains(got.stdout, "tmux not found") {
		t.Errorf("exit %d, stdout:\n%s", got.code, got.stdout)
	}
}
