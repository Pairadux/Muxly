// Package tmuxtest gives tests a real tmux server on a private socket, so they can never
// reach the server of whoever runs them.
package tmuxtest

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// waitLimit is how long Eventually waits. It's generous because a race-enabled build on
// a busy CI machine can be slow to start, and a passing test doesn't wait this long.
const waitLimit = 20 * time.Second

// Server is a private tmux server, which starts when the first command reaches it.
//
// Code under test finds it through a "tmux" wrapper that New writes, which pins every
// call to the private socket and logs it. That also holds when the test itself runs
// inside tmux, because -S beats $TMUX.
type Server struct {
	t        *testing.T
	realTmux string
	wrapper  string
	socket   string
	terminal string
	logFile  string
	env      []string
}

// Require returns the path of a tool the test can't run without. A missing tool skips
// the test on a developer's machine but fails it in CI, so CI can't pass by not running
// these tests at all.
func Require(t *testing.T, tool string) string {
	t.Helper()
	path, err := exec.LookPath(tool)
	if err == nil {
		return path
	}
	if os.Getenv("CI") != "" {
		t.Fatalf("%s must be installed in CI: %v", tool, err)
	}
	t.Skipf("%s is not installed", tool)
	return ""
}

// New prepares a private server and writes its "tmux" wrapper into binDir. Put binDir on
// PATH to make the code under test use it.
func New(t *testing.T, binDir string) *Server {
	t.Helper()
	realTmux := Require(t, "tmux")

	// Unix socket paths are limited to about 100 bytes, which t.TempDir and a long
	// $TMPDIR can exceed, so these live directly under /tmp.
	dir, err := os.MkdirTemp("/tmp", "mx")
	if err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(dir, "home")
	if err := os.Mkdir(home, 0o755); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		t:        t,
		realTmux: realTmux,
		wrapper:  filepath.Join(binDir, "tmux"),
		socket:   filepath.Join(dir, "s"),
		terminal: filepath.Join(dir, "o"),
		logFile:  filepath.Join(dir, "calls"),
		env: []string{
			"PATH=" + binDir,
			"HOME=" + home,
			"SHELL=/bin/sh",
			"TERM=xterm-256color",
			"LANG=C.UTF-8",
		},
	}
	t.Cleanup(func() {
		s.stop()
		// A server can take a moment to remove its socket as it exits.
		_ = os.RemoveAll(dir)
	})
	s.writeWrapper("")
	return s
}

// FailOn makes the wrapper refuse a subcommand with a fixed message, to test how
// failures are handled at a chosen step. An empty subcommand removes the failure.
func (s *Server) FailOn(subcommand string) {
	s.t.Helper()
	s.writeWrapper(subcommand)
}

// InjectedFailure is the message tmux seems to print for a subcommand given to FailOn.
const InjectedFailure = "injected failure"

func (s *Server) writeWrapper(failOn string) {
	s.t.Helper()
	script := fmt.Sprintf(`#!/bin/sh
sub=$1
[ "$sub" = "-u" ] && sub=$2
printf '%%s\n' "$*" >> %[1]q
if [ "$sub" = %[2]q ]; then
	echo %[3]q >&2
	exit 1
fi
exec %[4]q -f /dev/null -S %[5]q "$@"
`, s.logFile, failOn, InjectedFailure, s.realTmux, s.socket)
	if err := os.WriteFile(s.wrapper, []byte(script), 0o755); err != nil {
		s.t.Fatal(err)
	}
}

// Socket returns the path of the server's socket.
func (s *Server) Socket() string {
	return s.socket
}

// Env returns the sealed environment the server's commands run in.
func (s *Server) Env() []string {
	return s.env
}

func (s *Server) command(socket string, args ...string) *exec.Cmd {
	cmd := exec.Command(s.realTmux, append([]string{"-f", "/dev/null", "-S", socket}, args...)...)
	// The first command starts the server, which keeps this environment for its panes,
	// so nothing from the developer's shell or profile may leak in.
	cmd.Env = s.env
	return cmd
}

// Try runs a tmux command on the server, bypassing the wrapper, and returns its output.
func (s *Server) Try(args ...string) (string, error) {
	out, err := s.command(s.socket, args...).CombinedOutput()
	return strings.TrimRight(string(out), "\n"), err
}

// Run is Try for commands that must work. A failure fails the test.
func (s *Server) Run(args ...string) string {
	s.t.Helper()
	out, err := s.Try(args...)
	if err != nil {
		s.t.Fatalf("tmux %s: %v: %s", strings.Join(args, " "), err, out)
	}
	return out
}

func (s *Server) stop() {
	// Failing here means that server never started, which is fine.
	_ = s.command(s.socket, "kill-server").Run()
	_ = s.command(s.terminal, "kill-server").Run()
}

// NewSession creates a detached session and returns its $id.
func (s *Server) NewSession(name string) string {
	s.t.Helper()
	return s.Run("new-session", "-d", "-P", "-F", "#{session_id}", "-s", name, "-x", "100", "-y", "30")
}

// Sessions returns the names of the running sessions, or nil if no server is running.
func (s *Server) Sessions() []string {
	out, err := s.Try("list-sessions", "-F", "#{session_name}")
	if err != nil || out == "" {
		return nil
	}
	return strings.Split(out, "\n")
}

// HasServer reports whether the server is running.
func (s *Server) HasServer() bool {
	_, err := s.Try("list-sessions")
	return err == nil
}

// Pane returns the $TMUX_PANE value of the active pane in a session.
func (s *Server) Pane(sessionID string) string {
	s.t.Helper()
	return s.Run("display-message", "-p", "-t", sessionID, "#{pane_id}")
}

// Inside returns what makes a process believe it's running in a session's active pane,
// as KEY=value pairs.
func (s *Server) Inside(sessionID string) []string {
	s.t.Helper()
	return []string{"TMUX=" + s.socket + ",1,0", "TMUX_PANE=" + s.Pane(sessionID)}
}

// ClientSession returns the name of the session the attached client is on, or "" if no
// client is attached.
func (s *Server) ClientSession() string {
	out, err := s.Try("list-clients", "-F", "#{client_session}")
	if err != nil {
		return ""
	}
	return out
}

// Calls returns the tmux commands the code under test ran, one per line.
func (s *Server) Calls() []string {
	s.t.Helper()
	data, err := os.ReadFile(s.logFile)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		s.t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// Attach attaches a real client to a session and waits until tmux lists it.
func (s *Server) Attach(sessionID string) {
	s.t.Helper()
	s.InTerminal(s.realTmux, "-f", "/dev/null", "-S", s.socket, "attach-session", "-t", sessionID)
	s.Eventually("a client to attach", func() bool { return s.ClientSession() != "" })
}

// InTerminal runs a command on a real terminal. A tmux client or fzf needs one, and Go's
// standard library can't create one, so this uses a pane of a second private server.
// tmux marks its panes through the environment, which env removes so the command is
// outside tmux as far as it can tell.
func (s *Server) InTerminal(args ...string) {
	s.t.Helper()
	full := append([]string{"new-session", "-d", "-x", "100", "-y", "30", "--", "/usr/bin/env", "-u", "TMUX", "-u", "TMUX_PANE"}, args...)
	if out, err := s.command(s.terminal, full...).CombinedOutput(); err != nil {
		s.t.Fatalf("starting a terminal: %v: %s", err, out)
	}
}

// TerminalText returns what the terminal from InTerminal currently shows.
func (s *Server) TerminalText() string {
	out, err := s.command(s.terminal, "capture-pane", "-p").CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}

// PaneText returns what a session's active pane currently shows.
func (s *Server) PaneText(sessionID string) string {
	out, err := s.Try("capture-pane", "-p", "-t", sessionID)
	if err != nil {
		return ""
	}
	return out
}

// Eventually waits for check to pass, and fails the test if it never does. Use it for
// anything that happens asynchronously: real clients and panes take a moment to react.
func (s *Server) Eventually(what string, check func() bool) {
	s.t.Helper()
	deadline := time.Now().Add(waitLimit)
	for !check() {
		if time.Now().After(deadline) {
			s.t.Fatalf("timed out waiting for %s\nsessions: %q\nclient on: %q", what, s.Sessions(), s.ClientSession())
		}
		time.Sleep(20 * time.Millisecond)
	}
}
