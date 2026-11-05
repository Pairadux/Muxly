//go:build unix

package tmux

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/tmuxtest"
)

// sessionNamed builds what CreateAndSwitchSession takes, with one window per given name.
func sessionNamed(name, path string, windows ...string) models.Session {
	layout := models.SessionLayout{}
	for _, w := range windows {
		layout.Windows = append(layout.Windows, models.Window{Name: w})
	}
	return models.Session{Name: name, Path: path, Layout: layout}
}

func TestListSessionsReportsWhatTmuxHas(t *testing.T) {
	srv := privateServer(t)
	dir := t.TempDir()
	id := srv.Run("new-session", "-d", "-P", "-F", "#{session_id}", "-s", "foo", "-c", dir)

	got, err := ListSessions()

	if err != nil {
		t.Fatal(err)
	}
	if want := []Session{{ID: id, Name: "foo", Path: dir}}; !slices.Equal(got, want) {
		t.Errorf("ListSessions() = %+v, want %+v", got, want)
	}
}

// tmux reports a missing server in two ways. Neither is a failure for muxly.
func TestNoServerMeansNoSessions(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*tmuxtest.Server)
	}{
		{name: "the socket doesn't exist", setup: func(*tmuxtest.Server) {}},
		{
			name: "the socket is left over from a server that died",
			setup: func(srv *tmuxtest.Server) {
				srv.NewSession("a")
				pid := srv.Run("display-message", "-p", "#{pid}")
				srv.Run("run-shell", "-b", "kill -9 "+pid)
				srv.Eventually("the server to die", func() bool { return !srv.HasServer() })
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := privateServer(t)
			tt.setup(srv)

			got, err := ListSessions()

			if err != nil || got != nil {
				t.Errorf("ListSessions() = %+v, %v: want no sessions and no error", got, err)
			}
			if IsTmuxServerRunning() {
				t.Error("IsTmuxServerRunning() = true")
			}
		})
	}
}

// Outside tmux with a locale that isn't UTF-8, tmux would otherwise replace the tabs in
// sessionFormat and every non-ASCII character with "_".
func TestSessionsParseWithoutAUTF8Locale(t *testing.T) {
	srv := privateServer(t)
	id := srv.NewSession("é-🎉")
	t.Setenv("LANG", "C")
	t.Setenv("LC_ALL", "C")
	t.Setenv("LC_CTYPE", "C")

	got, err := ListSessions()
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].ID != id || got[0].Name != "é-🎉" {
		t.Errorf("ListSessions() = %+v", got)
	}
}

func TestCurrentSession(t *testing.T) {
	srv := privateServer(t)

	if got, inside := CurrentSession(); inside {
		t.Errorf("outside tmux, CurrentSession() = %+v", got)
	}

	srv.NewSession("other")
	id := srv.NewSession("mine")
	runInside(t, srv, id)

	got, inside := CurrentSession()

	if !inside || got.ID != id || got.Name != "mine" {
		t.Errorf("CurrentSession() = %+v, %v", got, inside)
	}
}

// tmux would match "foo" to foobar by prefix if it were given the name.
func TestAPrefixOfASessionNameCreatesANewSession(t *testing.T) {
	srv := privateServer(t)
	shell := srv.NewSession("shell")
	srv.NewSession("foobar")
	srv.Attach(shell)
	runInside(t, srv, shell)

	dir := t.TempDir()
	if err := CreateAndSwitchSession(sessionNamed("foo", dir, "main")); err != nil {
		t.Fatal(err)
	}

	if got, want := srv.Sessions(), []string{"foo", "foobar", "shell"}; !slices.Equal(got, want) {
		t.Errorf("sessions = %q, want %q", got, want)
	}
	srv.Eventually("the client to reach foo", func() bool { return srv.ClientSession() == "foo" })
}

// Each of these names makes tmux act on the wrong thing, or nothing, when it's given as a
// target. The name must open one session, and opening it again must find that session.
func TestAwkwardNamesOpenOneSessionAndFindItAgain(t *testing.T) {
	tests := []struct {
		name   string
		stored string
	}{
		{name: "x.y/src", stored: "x_y/src"},
		{name: ".conf/app", stored: "_conf/app"},
		{name: "foo.bar", stored: "foo_bar"},
		{name: "a:b", stored: "a_b"},
		{name: "$dollar", stored: "$dollar"},
		{name: "%pct", stored: "%pct"},
		{name: "=eq", stored: "=eq"},
		{name: "@at", stored: "@at"},
		{name: "é-🎉", stored: "é-🎉"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := privateServer(t)
			shell := srv.NewSession("shell")
			srv.Attach(shell)
			runInside(t, srv, shell)
			want := []string{"shell", tt.stored}
			slices.Sort(want)

			for _, attempt := range []string{"opening", "opening again"} {
				if err := CreateAndSwitchSession(sessionNamed(tt.name, "/tmp", "main", "term")); err != nil {
					t.Fatalf("%s: %v", attempt, err)
				}
				if got := srv.Sessions(); !slices.Equal(got, want) {
					t.Fatalf("%s: sessions = %q, want %q", attempt, got, want)
				}
				srv.Eventually("the client to arrive", func() bool { return srv.ClientSession() == tt.stored })
				srv.Run("switch-client", "-t", shell)
				srv.Eventually("the client to return", func() bool { return srv.ClientSession() == "shell" })
			}

			for _, call := range srv.Calls() {
				if strings.Contains(call, "-t "+tt.name) || strings.Contains(call, "-t "+tt.stored) {
					t.Errorf("a command targeted the session by name: %q", call)
				}
			}
		})
	}
}

// tmux 3.7a onward can hold a dot in a name, so a session made outside muxly may have one.
func TestAnExactNameWinsOverTheNameMuxlyWouldUse(t *testing.T) {
	srv := privateServer(t)
	shell := srv.NewSession("shell")
	dotted := srv.NewSession("foo.bar")
	stored := srv.Run("display-message", "-p", "-t", dotted, "#{session_name}")
	if stored != "foo.bar" {
		t.Skipf("this tmux stores foo.bar as %q", stored)
	}
	srv.NewSession("foo_bar")
	srv.Attach(shell)
	runInside(t, srv, shell)

	if err := CreateAndSwitchSession(sessionNamed("foo.bar", "/tmp", "main")); err != nil {
		t.Fatal(err)
	}

	srv.Eventually("the client to reach foo.bar", func() bool { return srv.ClientSession() == "foo.bar" })
}

func TestSwitchKeepsTheActiveWindow(t *testing.T) {
	for _, baseIndex := range []string{"0", "1"} {
		t.Run("base-index "+baseIndex, func(t *testing.T) {
			srv := privateServer(t)
			shell := srv.NewSession("shell")
			srv.Run("set-option", "-g", "base-index", baseIndex)
			target := srv.Run("new-session", "-d", "-P", "-F", "#{session_id}", "-s", "target", "-n", "first")
			srv.Run("new-window", "-t", target, "-n", "second")
			srv.Run("new-window", "-t", target, "-n", "third")
			srv.Attach(shell)
			runInside(t, srv, shell)

			if err := SwitchToSession(Session{ID: target, Name: "target"}); err != nil {
				t.Fatal(err)
			}

			srv.Eventually("the client to reach target", func() bool { return srv.ClientSession() == "target" })
			if got := srv.Run("display-message", "-p", "-t", target, "#{window_name}"); got != "third" {
				t.Errorf("active window is %q, want third", got)
			}
		})
	}
}

func TestCreatedSessionHasTheLayoutInTheRightPlace(t *testing.T) {
	srv := privateServer(t)
	dir := t.TempDir()
	session := models.Session{
		Name: "proj",
		Path: dir,
		Layout: models.SessionLayout{Windows: []models.Window{
			{Name: "editor"},
			{Name: "run", Cmd: "true"},
		}},
	}

	created, err := createSession(session)
	if err != nil {
		t.Fatal(err)
	}

	if created.Name != "proj" || created.Path != dir || !strings.HasPrefix(created.ID, "$") {
		t.Errorf("createSession() = %+v", created)
	}
	if got := windowsOf(srv, "proj"); !slices.Equal(got, []string{"editor", "run"}) {
		t.Errorf("windows = %q", got)
	}
	want, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	srv.Eventually("every window's shell to start in the session's path", func() bool {
		paths := srv.Run("list-windows", "-t", created.ID, "-F", "#{pane_current_path}")
		return paths == want+"\n"+want
	})
}

func TestAFailedWindowLeavesNoHalfMadeSession(t *testing.T) {
	srv := privateServer(t)
	srv.NewSession("shell")
	srv.FailOn("new-window")

	_, err := createSession(sessionNamed("proj", "/tmp", "one", "two"))

	if err == nil || !strings.Contains(err.Error(), tmuxtest.InjectedFailure) {
		t.Errorf("err = %v, want tmux's message", err)
	}
	if got := srv.Sessions(); !slices.Equal(got, []string{"shell"}) {
		t.Errorf("sessions = %q, want only shell", got)
	}
}

func TestLayoutWithNoWindowsCreatesNothing(t *testing.T) {
	srv := privateServer(t)

	if _, err := createSession(sessionNamed("proj", "/tmp")); err == nil {
		t.Error("want an error")
	}
	if got := srv.Calls(); got != nil {
		t.Errorf("tmux was called: %q", got)
	}
}

func TestKillSessionKillsOnlyThatSession(t *testing.T) {
	srv := privateServer(t)
	srv.NewSession("foobar")
	foo := srv.NewSession("foo")

	if err := KillSession(Session{ID: foo, Name: "foo"}); err != nil {
		t.Fatal(err)
	}

	if got := srv.Sessions(); !slices.Equal(got, []string{"foobar"}) {
		t.Errorf("sessions = %q, want only foobar", got)
	}
}

func TestFailedCommandsCarryTmuxsOwnMessage(t *testing.T) {
	srv := privateServer(t)
	srv.NewSession("a")

	err := KillSession(Session{ID: "$999", Name: "gone"})

	if err == nil || !strings.Contains(err.Error(), "$999") || strings.Contains(err.Error(), "exit status") {
		t.Errorf("err = %v, want tmux's message about $999", err)
	}
	var cmdErr *commandError
	if !errors.As(err, &cmdErr) {
		t.Errorf("err is a %T", err)
	}
}

func TestDefaultTemplatePathIsExpanded(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "tilde path", path: "~/x", want: "x"},
		{name: "tilde", path: "~", want: ""},
		{name: "no path opens in home", path: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := privateServer(t)
			want := filepath.Join(os.Getenv("HOME"), tt.want)
			if err := os.MkdirAll(want, 0o755); err != nil {
				t.Fatal(err)
			}
			old := srv.NewSession("current")
			srv.Attach(old)
			runInside(t, srv, old)

			if err := ReplaceLastSession(replaceConfig(tt.path), Session{ID: old, Name: "current"}); err != nil {
				t.Fatal(err)
			}

			sessions, err := ListSessions()
			if err != nil {
				t.Fatal(err)
			}
			created, found := FindSession(sessions, "default")
			if !found || created.Path != want {
				t.Errorf("session = %+v, want its path to be %q", created, want)
			}
		})
	}
}

// A path that can't be resolved must fail before anything has changed.
func TestUnresolvableTemplatePathKillsNothing(t *testing.T) {
	srv := privateServer(t)
	old := srv.NewSession("default")
	srv.Attach(old)
	runInside(t, srv, old)

	err := ReplaceLastSession(replaceConfig("./relative"), Session{ID: old, Name: "default"})

	if err == nil || !strings.Contains(err.Error(), "relative") {
		t.Errorf("err = %v, want it to name the path", err)
	}
	if got := srv.Sessions(); !slices.Equal(got, []string{"default"}) {
		t.Errorf("sessions = %q", got)
	}
	if got := srv.Calls(); got != nil {
		t.Errorf("tmux was called: %q", got)
	}
}
