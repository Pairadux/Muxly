//go:build unix

package cmd

import (
	"slices"
	"strings"
	"testing"
)

// insideTmux makes muxly believe it's running in session current, with sessions alive.
func (c *cli) insideTmux(current string, sessions ...string) *cli {
	return c.setEnv(
		"TMUX=/fake/socket,1,0",
		"FAKE_CURRENT="+current,
		"FAKE_SESSIONS="+strings.Join(sessions, ","),
	)
}

// alwaysKillConfig skips the last-session prompt, which needs a real terminal.
const alwaysKillConfig = "entry_dirs:\n  - path: /tmp\n" + `templates:
  - name: default
    default: true
    windows:
      - name: main
settings:
  always_kill_on_last_session: true
`

func TestKillOutsideTmuxNeverOffersToKillServer(t *testing.T) {
	for _, args := range [][]string{{"kill"}, {"kill", "precious"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig).setEnv("FAKE_SESSIONS=precious")

			got := c.run(args...)

			if got.code == 0 {
				t.Error("want a non-zero exit")
			}
			if !strings.Contains(got.stderr, "muxly kill -k") {
				t.Errorf("stderr %q should point at -k", got.stderr)
			}
			if c.tmuxCalled("kill-server") || c.tmuxCalled("kill-session") {
				t.Errorf("something was killed: %q", c.log("tmux"))
			}
		})
	}
}

func TestKillServerFlagKillsServer(t *testing.T) {
	c := newCLI(t).withFakes().writeConfig(validConfig).setEnv("FAKE_SESSIONS=precious")

	got := c.run("kill", "-k")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if !c.tmuxCalled("kill-server") {
		t.Errorf("kill-server wasn't called: %q", c.log("tmux"))
	}
}

func TestKillWithNoServer(t *testing.T) {
	c := newCLI(t).withFakes().writeConfig(validConfig)

	got := c.run("kill")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if c.tmuxCalled("kill-server") || c.tmuxCalled("kill-session") {
		t.Errorf("something was killed: %q", c.log("tmux"))
	}
}

func TestKillLastSessionDoesNotFallThroughToPicker(t *testing.T) {
	for _, args := range [][]string{{"kill"}, {"kill", "only"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(alwaysKillConfig).insideTmux("only", "only")

			got := c.run(args...)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if !c.tmuxCalled("kill-server") {
				t.Errorf("kill-server wasn't called: %q", c.log("tmux"))
			}
			if c.tmuxCalled("switch-client") {
				t.Errorf("tried to switch: %q", c.log("tmux"))
			}
			if c.log("fzf") != nil {
				t.Error("the picker opened")
			}
		})
	}
}

// Without a terminal the prompt can't be answered, and that must not count as a "no",
// which would kill the server.
func TestKillLastSessionPromptFailureKillsNothing(t *testing.T) {
	c := newCLI(t).withFakes().writeConfig(validConfig).insideTmux("only", "only")

	got := c.run("kill")

	if got.code == 0 {
		t.Error("want a non-zero exit")
	}
	if c.tmuxCalled("kill-server") || c.tmuxCalled("kill-session") {
		t.Errorf("something was killed: %q", c.log("tmux"))
	}
}

func TestKillSwitchesThenKillsCurrent(t *testing.T) {
	tests := []struct {
		name string
		args []string
		pick string
	}{
		{name: "named session", args: []string{"kill", "other"}},
		{name: "picked session", args: []string{"kill"}, pick: "other"},
		{name: "naming the current session means pick one", args: []string{"kill", "current"}, pick: "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig).
				insideTmux("current", "current", "other").setEnv("FAKE_PICK=" + tt.pick)

			got := c.run(tt.args...)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			calls := c.log("tmux")
			switched := slices.Index(calls, "switch-client -t other:0")
			killed := slices.Index(calls, "kill-session -t current")
			if switched == -1 || killed == -1 || switched > killed {
				t.Errorf("want a switch to other, then the kill: %q", calls)
			}
			if opened := c.log("fzf") != nil; opened != (tt.pick != "") {
				t.Errorf("picker opened = %v", opened)
			}
		})
	}
}

// tmux matches -t by prefix, so "defau" would otherwise find the current session, and
// killing it after switching to it would end the server.
func TestKillRejectsNamesThatAreNotExactlyAnotherSession(t *testing.T) {
	for _, name := range []string{"defau", "nope"} {
		t.Run(name, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig).insideTmux("default", "default")

			got := c.run("kill", name)

			if got.code == 0 || !strings.Contains(got.stderr, name) {
				t.Errorf("exit %d, stderr %q", got.code, got.stderr)
			}
			if c.tmuxCalled("kill-server") || c.tmuxCalled("kill-session") || c.tmuxCalled("switch-client") {
				t.Errorf("unexpected tmux calls: %q", c.log("tmux"))
			}
		})
	}
}

func TestKillPickerCancelKillsNothing(t *testing.T) {
	c := newCLI(t).withFakes().writeConfig(validConfig).
		insideTmux("current", "current", "other").setEnv("FAKE_FZF_EXIT=130")

	got := c.run("kill")

	if got.code != 0 || got.stdout != "" || got.stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q: want a silent exit 0", got.code, got.stdout, got.stderr)
	}
	if c.tmuxCalled("kill-session") || c.tmuxCalled("kill-server") {
		t.Errorf("something was killed: %q", c.log("tmux"))
	}
}
