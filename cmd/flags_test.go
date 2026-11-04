//go:build unix

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// walkCommands calls fn on root and every command below it.
func walkCommands(root *cobra.Command, fn func(*cobra.Command)) {
	fn(root)
	for _, sub := range root.Commands() {
		walkCommands(sub, fn)
	}
}

// Cobra panics when a command's flags and the persistent flags it inherits use the same
// shorthand for different flags. It happens every time that command runs, even with
// --help, so walking the tree finds the crash here instead of in use.
func TestNoCommandHasConflictingShorthands(t *testing.T) {
	rootCmd.InitDefaultCompletionCmd()
	rootCmd.InitDefaultHelpCmd()

	walkCommands(rootCmd, func(c *cobra.Command) {
		t.Run(c.CommandPath(), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("merging flags panicked: %v", r)
				}
			}()
			c.InitDefaultHelpFlag()
			c.InitDefaultVersionFlag()
			c.LocalFlags()
			c.InheritedFlags()
		})
	})
}

func TestOnlyKillHasKillServerFlag(t *testing.T) {
	walkCommands(rootCmd, func(c *cobra.Command) {
		c.LocalFlags()
		if c.Flags().Lookup("kill-server") != nil && c != killCmd {
			t.Errorf("%q accepts --kill-server", c.CommandPath())
		}
	})
	if killCmd.Flags().ShorthandLookup("k") == nil {
		t.Error("kill does not accept -k")
	}
}

func TestKillShorthandRejectedOutsideKill(t *testing.T) {
	for _, args := range [][]string{{"-k"}, {"switch", "-k"}, {"doctor", "-k"}, {"config", "list", "-k"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)

			got := c.run(args...)

			if got.code == 0 || !strings.Contains(got.stderr, "unknown shorthand flag") {
				t.Errorf("exit %d, stderr %q: want an unknown flag error", got.code, got.stderr)
			}
			if c.tmuxCalled("kill-server") {
				t.Error("kill-server was called")
			}
		})
	}
}

func TestConfigRemoveEntry(t *testing.T) {
	tests := []struct {
		name         string
		entry        func(proj string) string
		flags        []string
		hasMuxlyFile bool
		wantKeptFile bool
	}{
		{name: "absolute path", entry: func(proj string) string { return proj }},
		{name: "tilde path in the config", entry: func(string) string { return "~/proj" }},
		{
			name:         "-k keeps the .muxly file",
			entry:        func(string) string { return "~/proj" },
			flags:        []string{"-k"},
			hasMuxlyFile: true,
			wantKeptFile: true,
		},
		{
			name:         "-d deletes the .muxly file",
			entry:        func(string) string { return "~/proj" },
			flags:        []string{"-d"},
			hasMuxlyFile: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t)
			proj := filepath.Join(c.home, "proj")
			c.writeConfig("entry_dirs:\n  - path: " + tt.entry(proj) + "\n  - path: /keep\n" + validTemplates)
			muxlyFile := filepath.Join(proj, ".muxly")
			if tt.hasMuxlyFile {
				c.writeFile(muxlyFile, "windows:\n  - name: main\n", 0o644)
			}

			args := append([]string{"config", "remove", "entry"}, tt.flags...)
			got := c.run(append(args, proj)...)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if cfg := c.readConfig(); strings.Contains(cfg, "proj") || !strings.Contains(cfg, "/keep") {
				t.Errorf("config after removal:\n%s", cfg)
			}
			if tt.hasMuxlyFile {
				_, err := os.Stat(muxlyFile)
				if kept := err == nil; kept != tt.wantKeptFile {
					t.Errorf(".muxly kept = %v, want %v", kept, tt.wantKeptFile)
				}
			}
		})
	}
}

func TestConfigRemoveEntryNotConfigured(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)

	got := c.run("config", "remove", "entry", "/not/there")

	if got.code == 0 || !strings.Contains(got.stderr, "not in entry_dirs") {
		t.Errorf("exit %d, stderr %q", got.code, got.stderr)
	}
}
