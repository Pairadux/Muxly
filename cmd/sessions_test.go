//go:build unix

package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/tmuxtest"
)

// configWithEntries returns a valid config whose entry_dirs are the given paths.
func configWithEntries(paths ...string) string {
	var entries strings.Builder
	entries.WriteString("entry_dirs:\n")
	for _, path := range paths {
		entries.WriteString("  - path: " + path + "\n")
	}
	return entries.String() + validTemplates
}

func (c *cli) mkdirs(rel ...string) []string {
	c.t.Helper()
	paths := make([]string, len(rel))
	for i, r := range rel {
		paths[i] = filepath.Join(c.home, r)
		if err := os.MkdirAll(paths[i], 0o755); err != nil {
			c.t.Fatal(err)
		}
	}
	return paths
}

// targetsOnlyByID fails the test if any tmux command targeted something other than a $id.
func targetsOnlyByID(t *testing.T, srv *tmuxtest.Server) {
	t.Helper()
	id := regexp.MustCompile(`^\$[0-9]+$`)
	for _, call := range srv.Calls() {
		args := strings.Fields(call)
		for i, arg := range args {
			if arg == "-t" && i+1 < len(args) && !id.MatchString(args[i+1]) {
				t.Errorf("a command didn't target a $id: %q", call)
			}
		}
	}
}

// The whole path, with real fzf on a real terminal: type a name, press Enter, get that
// session. tmux alone would take "foo" to mean the running foobar.
func TestPickingADirectoryOpensItRatherThanASessionItPrefixes(t *testing.T) {
	c := newCLI(t)
	dirs := c.mkdirs("foo")
	c.writeConfig(configWithEntries(dirs...))
	srv := c.withRealTmux()
	c.withRealFzf()
	shell := srv.NewSession("shell")
	srv.NewSession("foobar")
	srv.Attach(shell)

	typeInPane(srv, shell, c.installMuxly())
	srv.Eventually("the picker to list foo", func() bool {
		return strings.Contains(srv.PaneText(shell), "[TMUX] foobar")
	})
	srv.Run("send-keys", "-t", shell, "-l", "^foo$")
	srv.Eventually("the filter to apply", func() bool {
		return !strings.Contains(srv.PaneText(shell), "[TMUX] foobar")
	})
	srv.Run("send-keys", "-t", shell, "Enter")

	srv.Eventually("foo to be created", func() bool { return slices.Contains(srv.Sessions(), "foo") })
	srv.Eventually("the client to reach foo", func() bool { return srv.ClientSession() == "foo" })
	if got := srv.Run("display-message", "-p", "-t", "=foo:", "#{session_path}"); got != dirs[0] {
		t.Errorf("foo is at %q, want %q", got, dirs[0])
	}
	if got, want := srv.Sessions(), []string{"foo", "foobar", "shell"}; !slices.Equal(got, want) {
		t.Errorf("sessions = %q, want %q", got, want)
	}
	targetsOnlyByID(t, srv)
}

// Each directory gets a picker name that tmux can't target by name. Picking it must open
// one session, and picking that session from the picker must then return to it. A name
// that muxly had to change also stays listed under the original, which must find it too.
func TestPickingAwkwardNamesOpensOneSessionAndFindsItAgain(t *testing.T) {
	tests := []struct {
		name   string
		dirs   []string
		pick   string
		stored string
		repick bool
	}{
		{name: "dot in a parent", dirs: []string{"x.y/src", "other/src"}, pick: "x.y/src", stored: "x_y/src", repick: true},
		{name: "leading dot in a parent", dirs: []string{".conf/app", "other/app"}, pick: ".conf/app", stored: "_conf/app", repick: true},
		{name: "dollar", dirs: []string{"$dollar"}, pick: "$dollar", stored: "_dollar", repick: true},
		{name: "percent", dirs: []string{"%pct"}, pick: "%pct", stored: "%pct"},
		{name: "equals", dirs: []string{"=eq"}, pick: "=eq", stored: "=eq"},
		{name: "at", dirs: []string{"@at"}, pick: "@at", stored: "@at"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t)
			for _, dir := range tt.dirs {
				c.mkdirs(filepath.Join("root", dir))
			}
			// From scan_dirs, because paths in the config have $VARIABLES expanded.
			c.writeConfig("scan_dirs:\n  - path: " + filepath.Join(c.home, "root") + "\n    depth: 2\n" + validTemplates)
			srv := c.withRealTmux()
			shell := srv.NewSession("shell")
			srv.Attach(shell)
			c.insideTmux(srv, shell)
			want := []string{"shell", tt.stored}
			slices.Sort(want)

			picks := []string{tt.pick, "[TMUX] " + tt.stored}
			if tt.repick {
				picks = append(picks, tt.pick)
			}
			for _, pick := range picks {
				c.setEnv("FAKE_PICK=" + pick)

				got := c.run()

				if got.code != 0 {
					t.Fatalf("picking %q: exit %d, stderr %q", pick, got.code, got.stderr)
				}
				if sessions := srv.Sessions(); !slices.Equal(sessions, want) {
					t.Fatalf("picking %q: sessions = %q, want %q", pick, sessions, want)
				}
				srv.Eventually("the client to arrive", func() bool { return srv.ClientSession() == tt.stored })
				srv.Run("switch-client", "-t", shell)
				srv.Eventually("the client to return", func() bool { return srv.ClientSession() == "shell" })
			}
			targetsOnlyByID(t, srv)
		})
	}
}

func TestSessionNameArgumentWithADot(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	shell := srv.NewSession("shell")
	srv.Attach(shell)
	c.insideTmux(srv, shell)

	for _, attempt := range []string{"first", "second"} {
		got := c.run("foo.bar")

		if got.code != 0 {
			t.Fatalf("%s: exit %d, stderr %q", attempt, got.code, got.stderr)
		}
		if sessions, want := srv.Sessions(), []string{"foo_bar", "shell"}; !slices.Equal(sessions, want) {
			t.Fatalf("%s: sessions = %q, want %q", attempt, sessions, want)
		}
	}
}

// The name from the picker has the prefix, and has to lose it to find the session. The
// session here was made outside muxly, with whatever name this version of tmux gave it:
// 3.2a stores this one as \$dollar. Either way, a $ can't be allowed to read as an id.
func TestPickingARunningSessionSwitchesToIt(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	shell := srv.NewSession("shell")
	id := srv.NewSession("$dollar")
	stored := srv.Run("display-message", "-p", "-t", id, "#{session_name}")
	srv.Attach(shell)
	c.insideTmux(srv, shell).setEnv("FAKE_PICK=[TMUX] " + stored)
	want := []string{"shell", stored}
	slices.Sort(want)

	got := c.run()

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	srv.Eventually("the client to arrive", func() bool { return srv.ClientSession() == stored })
	if sessions := srv.Sessions(); !slices.Equal(sessions, want) {
		t.Errorf("sessions = %q, want %q", sessions, want)
	}
}

func TestSwitchKeepsTheActiveWindow(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	shell := srv.NewSession("shell")
	target := srv.Run("new-session", "-d", "-P", "-F", "#{session_id}", "-s", "target", "-n", "first")
	srv.Run("new-window", "-t", target, "-n", "second")
	srv.Run("new-window", "-t", target, "-n", "third")
	srv.Attach(shell)
	c.insideTmux(srv, shell)

	got := c.run("switch", "target")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	srv.Eventually("the client to reach target", func() bool { return srv.ClientSession() == "target" })
	if window := srv.Run("display-message", "-p", "-t", target, "#{window_name}"); window != "third" {
		t.Errorf("active window is %q, want third", window)
	}
}

func TestSwitchNeedsTheExactName(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	shell := srv.NewSession("shell")
	srv.NewSession("foobar")
	srv.Attach(shell)
	c.insideTmux(srv, shell)

	got := c.run("switch", "foo")

	if got.code == 0 || !strings.Contains(got.stderr, `"foo"`) {
		t.Errorf("exit %d, stderr %q: want a failure naming foo", got.code, got.stderr)
	}
	if client := srv.ClientSession(); client != "shell" {
		t.Errorf("the client moved to %q", client)
	}
}

// A real client from outside tmux: attaching, leaving, and killing the server from inside
// all count as success, with nothing printed about them.
func TestAttachFromOutsideTmux(t *testing.T) {
	tests := []struct {
		name  string
		leave func(*tmuxtest.Server)
	}{
		{name: "detach", leave: func(srv *tmuxtest.Server) { srv.Run("detach-client") }},
		{name: "kill the session", leave: func(srv *tmuxtest.Server) { srv.Run("kill-session", "-t", "=foo") }},
		{name: "kill the server", leave: func(srv *tmuxtest.Server) { srv.Run("kill-server") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t).writeConfig(validConfig)
			srv := c.withRealTmux()
			muxly := c.installMuxly()

			srv.InTerminal("/bin/sh", "-c", muxly+` foo; echo "EXIT=$?"; exec /bin/cat`)

			srv.Eventually("the client to attach", func() bool { return srv.ClientSession() == "foo" })
			tt.leave(srv)
			srv.Eventually("muxly to exit", func() bool { return strings.Contains(srv.TerminalText(), "EXIT=") })
			text := srv.TerminalText()
			if !strings.Contains(text, "EXIT=0") || strings.Contains(text, "Error") {
				t.Errorf("want a clean exit, the terminal shows:\n%s", text)
			}
		})
	}
}

// Attaching needs a terminal. tmux's complaint about that has to reach the user.
func TestAttachFailureShowsTmuxsMessage(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	srv := c.withRealTmux()
	srv.NewSession("foo")

	got := c.run("foo")

	if got.code == 0 || strings.Contains(got.stderr, "exit status") || !strings.Contains(got.stderr, "terminal") {
		t.Errorf("exit %d, stderr %q: want tmux's message about the terminal", got.code, got.stderr)
	}
}
