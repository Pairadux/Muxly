//go:build unix

package cmd

import (
	"slices"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/tmuxtest"
)

func TestRuntimeErrorsDoNotPrintUsage(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cli)
		args  []string
		want  string
	}{
		{
			name:  "switch outside tmux",
			setup: func(c *cli) { c.withFakes().writeConfig(validConfig) },
			args:  []string{"switch"},
			want:  "not in Tmux",
		},
		{
			name:  "kill outside tmux",
			setup: func(c *cli) { c.withFakes().writeConfig(validConfig) },
			args:  []string{"kill"},
			want:  "not inside a tmux session",
		},
		{
			name:  "no config",
			setup: func(c *cli) { c.withFakes() },
			args:  []string{"switch"},
			want:  "no config file found",
		},
		{
			name:  "missing tools",
			setup: func(c *cli) { c.writeConfig(validConfig) },
			args:  []string{"switch"},
			want:  "missing required tools",
		},
		{
			name:  "config init over an existing file",
			setup: func(c *cli) { c.writeConfig(validConfig) },
			args:  []string{"config", "init"},
			want:  "already exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t)
			tt.setup(c)

			got := c.run(tt.args...)

			if got.code == 0 || !strings.Contains(got.stderr, tt.want) {
				t.Errorf("exit %d, stderr %q: want a failure mentioning %q", got.code, got.stderr, tt.want)
			}
			if strings.Contains(got.stderr, "Usage:") {
				t.Errorf("usage was printed:\n%s", got.stderr)
			}
		})
	}
}

func TestUsageMistakesStillPrintUsage(t *testing.T) {
	for _, args := range [][]string{{"--no-such-flag"}, {"a", "b"}, {"config", "add", "entry"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)

			got := c.run(args...)

			if got.code == 0 || !strings.Contains(got.stderr, "Usage:") {
				t.Errorf("exit %d, stderr %q: want usage", got.code, got.stderr)
			}
		})
	}
}

// fzf exits 1 when Enter is pressed with nothing matching, and 130 on Esc or Ctrl-C.
func TestClosingPickerIsSilent(t *testing.T) {
	commands := []struct {
		name string
		args []string
	}{
		{name: "root", args: nil},
		{name: "switch", args: []string{"switch"}},
		{name: "kill", args: []string{"kill"}},
	}

	for _, command := range commands {
		for _, exit := range []string{"1", "130"} {
			t.Run(command.name+" exit "+exit, func(t *testing.T) {
				c := newCLI(t).writeConfig(validConfig)
				srv := c.withRealTmux()
				current := srv.NewSession("current")
				srv.NewSession("other")
				c.insideTmux(srv, current).setEnv("FAKE_FZF_EXIT=" + exit)

				got := c.run(command.args...)

				if got.code != 0 || got.stdout != "" || got.stderr != "" {
					t.Errorf("exit %d, stdout %q, stderr %q: want a silent exit 0", got.code, got.stdout, got.stderr)
				}
				if c.log("fzf") == nil {
					t.Error("the picker never opened")
				}
				if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"current", "other"}) {
					t.Errorf("sessions changed: %q", sessions)
				}
			})
		}
	}
}

func TestPickerFailureIsReported(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	current := srv.NewSession("current")
	srv.NewSession("other")
	c.insideTmux(srv, current).setEnv("FAKE_FZF_EXIT=2")

	got := c.run("switch")

	if got.code == 0 || !strings.Contains(got.stderr, "fzf") {
		t.Errorf("exit %d, stderr %q: want an fzf error", got.code, got.stderr)
	}
}

// A failed tmux command must explain itself, not print "exit status 1". Each command here
// reaches tmux at a different point on its way to the failure.
func TestFailedTmuxCommandsShowTmuxsMessage(t *testing.T) {
	tests := []struct {
		name   string
		failOn string
		args   []string
	}{
		{name: "creating", failOn: "new-session", args: []string{"fresh"}},
		{name: "switching", failOn: "switch-client", args: []string{"switch", "other"}},
		{name: "killing a session", failOn: "kill-session", args: []string{"kill", "other"}},
		{name: "killing the server", failOn: "kill-server", args: []string{"kill", "-k"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t).writeConfig(validConfig)
			srv := c.withRealTmux()
			current := srv.NewSession("current")
			srv.NewSession("other")
			srv.Attach(current)
			c.insideTmux(srv, current)
			srv.FailOn(tt.failOn)

			got := c.run(tt.args...)

			if got.code == 0 || !strings.Contains(got.stderr, tmuxtest.InjectedFailure) {
				t.Errorf("exit %d, stderr %q: want tmux's message", got.code, got.stderr)
			}
			if strings.Contains(got.stderr, "exit status") {
				t.Errorf("stderr %q still has the exit status", got.stderr)
			}
			if strings.Contains(got.stderr, "Usage:") {
				t.Errorf("usage was printed:\n%s", got.stderr)
			}
		})
	}
}
