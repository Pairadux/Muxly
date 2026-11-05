//go:build unix

package cmd

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

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
			c := newCLI(t).writeConfig(validConfig)
			srv := c.withRealTmux()
			srv.NewSession("precious")

			got := c.run(args...)

			if got.code == 0 {
				t.Error("want a non-zero exit")
			}
			if !strings.Contains(got.stderr, "muxly kill -k") {
				t.Errorf("stderr %q should point at -k", got.stderr)
			}
			if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"precious"}) {
				t.Errorf("sessions = %q, want precious left alone", sessions)
			}
		})
	}
}

func TestKillServerFlagKillsServer(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	srv.NewSession("precious")

	got := c.run("kill", "-k")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if srv.HasServer() {
		t.Errorf("the server is still running: %q", srv.Sessions())
	}
}

func TestKillWithNoServer(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()

	got := c.run("kill")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if !strings.Contains(got.stdout, "No tmux server running") {
		t.Errorf("stdout = %q", got.stdout)
	}
	if calls := srv.Calls(); len(calls) != 1 {
		t.Errorf("want only the check for a server, got %q", calls)
	}
}

func TestKillLastSessionDoesNotFallThroughToPicker(t *testing.T) {
	for _, args := range [][]string{{"kill"}, {"kill", "only"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			c := newCLI(t).writeConfig(alwaysKillConfig)
			srv := c.withRealTmux()
			c.insideTmux(srv, srv.NewSession("only"))

			got := c.run(args...)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if srv.HasServer() {
				t.Errorf("the server is still running: %q", srv.Sessions())
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
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	c.insideTmux(srv, srv.NewSession("only"))

	got := c.run("kill")

	if got.code == 0 {
		t.Error("want a non-zero exit")
	}
	if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"only"}) {
		t.Errorf("sessions = %q, want only left alone", sessions)
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
			c := newCLI(t).writeConfig(validConfig).setEnv("FAKE_PICK=" + tt.pick)
			srv := c.withRealTmux()
			current := srv.NewSession("current")
			srv.NewSession("other")
			srv.Attach(current)
			c.insideTmux(srv, current)

			got := c.run(tt.args...)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"other"}) {
				t.Errorf("sessions = %q, want only other", sessions)
			}
			srv.Eventually("the client to reach other", func() bool { return srv.ClientSession() == "other" })
			if opened := c.log("fzf") != nil; opened != (tt.pick != "") {
				t.Errorf("picker opened = %v", opened)
			}
		})
	}
}

// Picking a session whose name holds syntax tmux would misread must kill the right one.
func TestKillWithAwkwardSessionNames(t *testing.T) {
	// The subtests are numbered because t.TempDir puts their names in the path of HOME,
	// where viper would read $dollar as a variable.
	for i, name := range []string{"$dollar", "%pct", "=eq", "@at"} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c := newCLI(t).writeConfig(validConfig)
			srv := c.withRealTmux()
			current := srv.NewSession(name)
			srv.NewSession("other")
			srv.Attach(current)
			c.insideTmux(srv, current)

			got := c.run("kill", "other")

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"other"}) {
				t.Errorf("sessions = %q, want only other", sessions)
			}
		})
	}
}

// tmux matches -t by prefix, so "defau" would otherwise find the current session, and
// killing it after switching to it would end the server.
func TestKillRejectsNamesThatAreNotExactlyAnotherSession(t *testing.T) {
	for _, name := range []string{"defau", "nope"} {
		t.Run(name, func(t *testing.T) {
			c := newCLI(t).writeConfig(validConfig)
			srv := c.withRealTmux()
			c.insideTmux(srv, srv.NewSession("default"))

			got := c.run("kill", name)

			if got.code == 0 || !strings.Contains(got.stderr, name) {
				t.Errorf("exit %d, stderr %q", got.code, got.stderr)
			}
			if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"default"}) {
				t.Errorf("sessions = %q, want default left alone", sessions)
			}
		})
	}
}

func TestKillPickerCancelKillsNothing(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig).setEnv("FAKE_FZF_EXIT=130")
	srv := c.withRealTmux()
	current := srv.NewSession("current")
	srv.NewSession("other")
	c.insideTmux(srv, current)

	got := c.run("kill")

	if got.code != 0 || got.stdout != "" || got.stderr != "" {
		t.Errorf("exit %d, stdout %q, stderr %q: want a silent exit 0", got.code, got.stdout, got.stderr)
	}
	if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"current", "other"}) {
		t.Errorf("sessions changed: %q", sessions)
	}
}

// From a real pane, killing the session ends the pane muxly runs in, so the kill has to
// be the last thing it does.
func TestKillFromARealPane(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig).setEnv("FAKE_PICK=other")
	srv := c.withRealTmux()
	current := srv.NewSession("current")
	srv.NewSession("other")
	srv.Attach(current)

	typeInPane(srv, current, c.installMuxly()+" kill")

	srv.Eventually("only other to remain", func() bool { return slices.Equal(srv.Sessions(), []string{"other"}) })
	srv.Eventually("the client to reach other", func() bool { return srv.ClientSession() == "other" })
}

// The prompt is real here, answered with real keys through a real terminal.
func TestKillLastSessionAnswerYesFromARealPane(t *testing.T) {
	for _, name := range []string{"default", "default-x", "other"} {
		t.Run(name, func(t *testing.T) {
			c := newCLI(t).writeConfig(validConfig)
			srv := c.withRealTmux()
			old := srv.NewSession(name)
			srv.Attach(old)

			typeInPane(srv, old, c.installMuxly()+" kill")
			srv.Eventually("the prompt", func() bool { return strings.Contains(srv.PaneText(old), "Create session") })
			srv.Run("send-keys", "-t", old, "y")

			srv.Eventually("a fresh session named default", func() bool {
				// Not Run: the old session may be giving up this name right now.
				id, err := srv.Try("display-message", "-p", "-t", "=default:", "#{session_id}")
				return err == nil && id != old && slices.Equal(srv.Sessions(), []string{"default"})
			})
			srv.Eventually("the client to reach it", func() bool { return srv.ClientSession() == "default" })
			if !srv.HasServer() {
				t.Error("the server died")
			}
		})
	}
}

func TestKillLastSessionAnswerNoKillsTheServer(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	old := srv.NewSession("only")
	srv.Attach(old)

	typeInPane(srv, old, c.installMuxly()+" kill")
	srv.Eventually("the prompt", func() bool { return strings.Contains(srv.PaneText(old), "Create session") })
	srv.Run("send-keys", "-t", old, "n")

	srv.Eventually("the server to end", func() bool { return !srv.HasServer() })
}
