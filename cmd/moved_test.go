//go:build unix

package cmd

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestMovedCommandsPointAtConfig(t *testing.T) {
	tests := []struct {
		args []string
		want string
	}{
		{args: []string{"add"}, want: "muxly config add"},
		{args: []string{"add", "scan", "/tmp"}, want: "muxly config add"},
		{args: []string{"add", "scan", "~/Dev", "--depth", "2", "--alias", "dev"}, want: "muxly config add"},
		{args: []string{"add", "--help"}, want: "muxly config add"},
		{args: []string{"remove"}, want: "muxly config remove"},
		{args: []string{"remove", "entry", "-k", "/tmp"}, want: "muxly config remove"},
		{args: []string{"remove", "scan", "/tmp"}, want: "muxly config remove"},
		{args: []string{"init"}, want: "muxly config init"},
		{args: []string{"edit"}, want: "muxly config edit"},
	}

	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)

			got := c.run(tt.args...)

			if got.code == 0 || !strings.Contains(got.stderr, tt.want) {
				t.Errorf("exit %d, stderr %q: want a failure mentioning %q", got.code, got.stderr, tt.want)
			}
			if strings.Contains(got.stderr, "Usage:") {
				t.Errorf("usage was printed:\n%s", got.stderr)
			}
			if calls := c.log("tmux"); calls != nil {
				t.Errorf("tmux was called: %q", calls)
			}
		})
	}
}

// Someone typing the old command on a fresh machine should still get the pointer.
func TestMovedCommandsNeedNoConfigOrTools(t *testing.T) {
	got := newCLI(t).run("add", "entry", ".")

	if got.code == 0 || !strings.Contains(got.stderr, "muxly config add") {
		t.Errorf("exit %d, stderr %q", got.code, got.stderr)
	}
}

func TestMovedCommandsAreHiddenFromHelp(t *testing.T) {
	got := newCLI(t).run("--help")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	for _, name := range []string{"add", "remove", "init", "edit"} {
		if regexp.MustCompile(`(?m)^  ` + name + `\s`).MatchString(got.stdout) {
			t.Errorf("%q is listed in help:\n%s", name, got.stdout)
		}
	}
}

var oldCommandPath = regexp.MustCompile(`muxly (add|remove)\b`)

func TestDocsUseCurrentCommandPaths(t *testing.T) {
	walkCommands(rootCmd, func(c *cobra.Command) {
		if match := oldCommandPath.FindString(c.Short + c.Long + c.Example); match != "" {
			t.Errorf("help for %q mentions %q", c.CommandPath(), match)
		}
	})

	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if matches := oldCommandPath.FindAllString(string(readme), -1); len(matches) > 0 {
		t.Errorf("README mentions the old commands %d times", len(matches))
	}
}
