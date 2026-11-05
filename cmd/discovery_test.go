//go:build unix

package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

// fastwalk reads a depth of 0 or less as no limit at all, so these must never reach it.
func TestDepthBelowOneIsRejected(t *testing.T) {
	for _, depth := range []string{"0", "-1"} {
		t.Run("root flag "+depth, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)

			got := c.run("--depth=" + depth)

			if got.code == 0 || !strings.Contains(got.stderr, "--depth must be at least 1") {
				t.Errorf("exit %d, stderr %q: want a depth error", got.code, got.stderr)
			}
			if c.log("fzf") != nil {
				t.Error("the picker opened")
			}
		})

		t.Run("config add scan flag "+depth, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)
			dir := c.mkdirs("dev")[0]

			got := c.run("config", "add", "scan", dir, "--depth="+depth)

			if got.code == 0 || !strings.Contains(got.stderr, "--depth must be at least 1") {
				t.Errorf("exit %d, stderr %q: want a depth error", got.code, got.stderr)
			}
			if strings.Contains(c.readConfig(), "dev") {
				t.Errorf("the config changed:\n%s", c.readConfig())
			}
		})

		t.Run("config file "+depth, func(t *testing.T) {
			c := newCLI(t).withFakes()
			c.writeConfig("scan_dirs:\n  - path: /tmp\n    depth: " + depth + "\n" + validTemplates)

			got := c.run()

			if got.code == 0 || !strings.Contains(got.stderr, "depth") {
				t.Errorf("exit %d, stderr %q: want a depth error", got.code, got.stderr)
			}
			if c.log("fzf") != nil {
				t.Error("the picker opened")
			}
		})
	}
}

func TestDepthFlagStillWorks(t *testing.T) {
	c := newCLI(t).withFakes()
	c.mkdirs("scan/one/two/three")
	c.writeConfig("scan_dirs:\n  - path: " + filepath.Join(c.home, "scan") + "\n" + validTemplates)

	got := c.run("--depth=2")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	picker := strings.Join(c.log("fzf"), "\n")
	if !strings.Contains(picker, "two") || strings.Contains(picker, "three") {
		t.Errorf("the picker offered:\n%s", picker)
	}
}
