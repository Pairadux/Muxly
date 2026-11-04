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
