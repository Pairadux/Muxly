//go:build unix

package tmux

import (
	"slices"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/tmuxtest"
)

// privateServer points this test's tmux commands at a private server, with nothing else
// on PATH, so they can't reach a real one.
func privateServer(t *testing.T) *tmuxtest.Server {
	t.Helper()
	bin := t.TempDir()
	srv := tmuxtest.New(t, bin)
	t.Setenv("PATH", bin)
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")
	t.Setenv("LANG", "C.UTF-8")
	return srv
}

// runInside makes this test's tmux commands act as if run from a session's active pane.
func runInside(t *testing.T, srv *tmuxtest.Server, sessionID string) {
	t.Helper()
	for _, pair := range srv.Inside(sessionID) {
		key, value, _ := strings.Cut(pair, "=")
		t.Setenv(key, value)
	}
}

// windowsOf returns a session's window names, in order.
func windowsOf(srv *tmuxtest.Server, session string) []string {
	return strings.Split(srv.Run("list-windows", "-t", "="+session, "-F", "#{window_name}"), "\n")
}

func replaceConfig(path string) *models.Config {
	return &models.Config{
		Templates: []models.SessionTemplate{{
			Name:    "default",
			Default: true,
			Path:    path,
			Windows: []models.Window{{Name: "main"}, {Name: "term"}},
		}},
	}
}

func TestReplaceLastSession(t *testing.T) {
	for _, current := range []string{
		"other",
		// Asking tmux whether "default" exists would match default-x by prefix, switching
		// to itself and then killing the last session.
		"default-x",
		// The old session has to give up the name first.
		"default",
	} {
		t.Run(current, func(t *testing.T) {
			srv := privateServer(t)
			old := srv.NewSession(current)
			srv.Attach(old)
			runInside(t, srv, old)

			err := ReplaceLastSession(replaceConfig("/tmp"), Session{ID: old, Name: current})
			if err != nil {
				t.Fatal(err)
			}

			if got := srv.Sessions(); !slices.Equal(got, []string{"default"}) {
				t.Errorf("sessions = %q, want only default", got)
			}
			if got := windowsOf(srv, "default"); !slices.Equal(got, []string{"main", "term"}) {
				t.Errorf("windows = %q, want the template's", got)
			}
			srv.Eventually("the client to move", func() bool { return srv.ClientSession() == "default" })
		})
	}
}

// Any failure before the old session is killed must leave things as they were.
func TestReplaceLastSessionFailureLeavesTheOldSession(t *testing.T) {
	tests := []struct {
		name    string
		current string
		failOn  string
	}{
		{name: "renaming", current: "default", failOn: "rename-session"},
		{name: "creating, after giving up the name", current: "default", failOn: "new-session"},
		{name: "adding a window", current: "other", failOn: "new-window"},
		{name: "adding a window, after giving up the name", current: "default", failOn: "new-window"},
		{name: "switching", current: "other", failOn: "switch-client"},
		{name: "switching, after giving up the name", current: "default", failOn: "switch-client"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := privateServer(t)
			old := srv.NewSession(tt.current)
			srv.Attach(old)
			runInside(t, srv, old)
			srv.FailOn(tt.failOn)

			err := ReplaceLastSession(replaceConfig("/tmp"), Session{ID: old, Name: tt.current})

			if err == nil || !strings.Contains(err.Error(), tmuxtest.InjectedFailure) {
				t.Errorf("err = %v, want tmux's message", err)
			}
			if got := srv.Sessions(); !slices.Equal(got, []string{tt.current}) {
				t.Errorf("sessions = %q, want only %q", got, tt.current)
			}
			if got := srv.Run("display-message", "-p", "-t", old, "#{session_name}"); got != tt.current {
				t.Errorf("the old session is now named %q", got)
			}
		})
	}
}

func TestReplaceLastSessionNoDefaultTemplate(t *testing.T) {
	srv := privateServer(t)

	if err := ReplaceLastSession(&models.Config{}, Session{ID: "$0", Name: "current"}); err == nil {
		t.Error("want an error")
	}
	if got := srv.Calls(); got != nil {
		t.Errorf("tmux was called: %q", got)
	}
}

// A template's name can hold characters tmux can't target, or changes on some versions.
// The failed new session must still be found and removed, leaving no orphan.
func TestReplaceLastSessionWithAnUntargetableTemplateNameLeavesNoOrphan(t *testing.T) {
	srv := privateServer(t)
	old := srv.NewSession("current")
	srv.Attach(old)
	runInside(t, srv, old)
	srv.FailOn("new-window")
	cfg := replaceConfig("/tmp")
	cfg.Templates[0].Name = "x.y"

	err := ReplaceLastSession(cfg, Session{ID: old, Name: "current"})

	if err == nil {
		t.Fatal("want an error")
	}
	if got := srv.Sessions(); !slices.Equal(got, []string{"current"}) {
		t.Errorf("sessions = %q, want only current", got)
	}
}
