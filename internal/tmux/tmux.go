// Package tmux runs all of muxly's tmux commands.
//
// Commands only ever target a session by its $id. Given a name, tmux also matches
// prefixes and patterns, and reads characters like "." and "$" as syntax, so it can act
// on the wrong session or none at all. Names are matched here in Go instead, against
// what ListSessions returns.
package tmux

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/utility"
)

// DefaultShell runs window commands when $SHELL isn't set.
const DefaultShell = "/bin/bash"

// sessionFormat is how muxly asks tmux to describe a session. tmux doesn't allow tabs in
// names, and the path comes last, so a tab in a path can't shift the other fields.
const sessionFormat = "#{session_id}\t#{session_name}\t#{session_path}"

// Session is a session running in tmux.
type Session struct {
	ID   string
	Name string
	Path string
}

// commandError is a failed tmux command, along with what tmux said about it.
type commandError struct {
	subcommand string
	stderr     string
	err        error
}

func (e *commandError) Error() string {
	if message := strings.TrimSpace(e.stderr); message != "" {
		return fmt.Sprintf("tmux %s: %s", e.subcommand, message)
	}
	return fmt.Sprintf("tmux %s: %v", e.subcommand, e.err)
}

func (e *commandError) Unwrap() error {
	return e.err
}

// run executes a tmux command and returns its output, without the trailing newline.
//
// Without -u, tmux replaces tabs and non-ASCII characters in its output with "_" when it
// runs outside tmux with a locale that isn't UTF-8, as it can under cron or a bare ssh.
func run(args ...string) (string, error) {
	cmd := exec.Command("tmux", append([]string{"-u"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &commandError{subcommand: args[0], stderr: stderr.String(), err: err}
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

// noServer reports whether err means that no tmux server is running. tmux exits 1 for
// every failure, so its message is the only way to tell. It says "no server running" when
// the socket is left over from an old server, and "error connecting" when there is none.
func noServer(err error) bool {
	var cmdErr *commandError
	if !errors.As(err, &cmdErr) {
		return false
	}
	return strings.HasPrefix(cmdErr.stderr, "no server running") ||
		strings.HasPrefix(cmdErr.stderr, "error connecting")
}

// parseSessions reads tmux's output in sessionFormat. It skips any line that doesn't fit,
// which takes a newline in a session's path.
//
// tmux stores a path exactly as it was given, with any trailing slash or "..", and a
// session started with no path at all has an empty one. Cleaning them here means a path
// can simply be compared, and an empty one matches nothing.
func parseSessions(output string) []Session {
	var sessions []Session
	for line := range strings.SplitSeq(output, "\n") {
		fields := strings.SplitN(line, "\t", 3)
		if len(fields) == 3 && strings.HasPrefix(fields[0], "$") {
			path := fields[2]
			if path != "" {
				path = filepath.Clean(path)
			}
			sessions = append(sessions, Session{ID: fields[0], Name: fields[1], Path: path})
		}
	}
	return sessions
}

// ListSessions returns the running sessions. No server just means no sessions.
func ListSessions() ([]Session, error) {
	output, err := run("list-sessions", "-F", sessionFormat)
	if noServer(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return parseSessions(output), nil
}

// FindSession returns the session with exactly the given name.
func FindSession(sessions []Session, name string) (Session, bool) {
	i := slices.IndexFunc(sessions, func(s Session) bool { return s.Name == name })
	if i < 0 {
		return Session{}, false
	}
	return sessions[i], true
}

// FindSessionAt returns the session started in the given clean, absolute path. If there
// are several, it returns the oldest. tmux lists sessions by name, and numbers them in
// order, so a shorter id is always older, and ids of one length sort as text.
func FindSessionAt(sessions []Session, path string) (Session, bool) {
	var oldest Session
	found := false
	for _, s := range sessions {
		if s.Path != path {
			continue
		}
		older := len(s.ID) < len(oldest.ID) || (len(s.ID) == len(oldest.ID) && s.ID < oldest.ID)
		if !found || older {
			oldest, found = s, true
		}
	}
	return oldest, found
}

// FreeSessionName returns a name no session has for a new session in path with the given
// label. If the label's name is taken, it adds the parent folder, and then a number.
func FreeSessionName(sessions []Session, label, path string) string {
	isFree := func(name string) bool {
		_, taken := FindSession(sessions, name)
		return !taken
	}

	name := SessionName(label)
	if isFree(name) {
		return name
	}
	withParent := SessionName(filepath.Base(filepath.Dir(path)) + "/" + label)
	if isFree(withParent) {
		return withParent
	}
	for n := 2; ; n++ {
		if numbered := fmt.Sprintf("%s-%d", name, n); isFree(numbered) {
			return numbered
		}
	}
}

// Without returns sessions, leaving out one session.
func Without(sessions []Session, session Session) []Session {
	return slices.DeleteFunc(slices.Clone(sessions), func(s Session) bool { return s.ID == session.ID })
}

// Names returns the names of sessions, in the same order.
func Names(sessions []Session) []string {
	names := make([]string, len(sessions))
	for i, s := range sessions {
		names[i] = s.Name
	}
	return names
}

// CurrentSession returns the session this process is running in. It reports false outside
// tmux, which includes $TMUX being left over from a server that has since gone.
func CurrentSession() (Session, bool) {
	if os.Getenv(constants.EnvTmux) == "" {
		return Session{}, false
	}

	output, err := run("display-message", "-p", sessionFormat)
	if err != nil {
		return Session{}, false
	}

	sessions := parseSessions(output)
	if len(sessions) != 1 {
		return Session{}, false
	}
	return sessions[0], true
}

// SwitchToSession moves to a session: it attaches a client from outside tmux, and switches
// the current client from inside. Switching to the bare $id keeps the session's active
// window, where naming a window would select that one.
func SwitchToSession(session Session) error {
	if os.Getenv(constants.EnvTmux) == "" {
		return attachToSession(session.ID)
	}

	_, err := run("switch-client", "-t", session.ID)
	return err
}

// IsTmuxServerRunning checks if a tmux server is currently running. If tmux fails for any
// other reason, there is still a server, and the next command will show what's wrong.
func IsTmuxServerRunning() bool {
	_, err := run("list-sessions")
	return !noServer(err)
}

// CreateAndSwitchSession moves to the session with the given name, creating it first if
// there isn't one. An existing session's exact name wins over the name muxly would give
// a new one, so a session named "foo.bar" is still found on a tmux that allows the dot.
func CreateAndSwitchSession(session models.Session) error {
	sessions, err := ListSessions()
	if err != nil {
		return err
	}

	for _, name := range []string{session.Name, SessionName(session.Name)} {
		if existing, found := FindSession(sessions, name); found {
			return SwitchToSession(existing)
		}
	}

	created, err := createSession(session)
	if err != nil {
		return fmt.Errorf("creating session: %w", err)
	}

	return SwitchToSession(created)
}

// fallbackSessionName names a session whose label has nothing usable in it, such as "🚀".
const fallbackSessionName = "session"

// SessionName returns the name muxly gives a new session with the given label: lowercase,
// with only letters, digits, "-" and "_", and each path component joined by "-".
//
// tmux stores many other characters differently from how it's given them, depending on
// the version, and then muxly can't find the session again. It also expands formats in a
// name, so "#{session_id}" would create a session with an empty name. Keeping to these
// characters means every version stores exactly the name muxly asked for.
func SessionName(label string) string {
	var parts []string
	for part := range strings.SplitSeq(strings.ToLower(label), "/") {
		if safe := safeNamePart(part); safe != "" {
			parts = append(parts, safe)
		}
	}
	if len(parts) == 0 {
		return fallbackSessionName
	}
	return strings.Join(parts, "-")
}

// safeNamePart makes one path component safe. "." becomes "_", as it always has in muxly,
// and anything else unusual becomes "-". Repeats collapse, and symbols at either end go.
func safeNamePart(part string) string {
	var b strings.Builder
	var previous rune
	for _, r := range part {
		switch {
		case r == '.':
			r = '_'
		case r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r):
		default:
			r = '-'
		}
		if (r == '-' || r == '_') && r == previous {
			continue
		}
		b.WriteRune(r)
		previous = r
	}
	return strings.Trim(b.String(), "-_")
}

// attachToSession runs a tmux client on this terminal, until it detaches or its session
// ends.
func attachToSession(id string) error {
	cmd := exec.Command("tmux", "attach-session", "-t", id)
	var stderr bytes.Buffer
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	// Killing the server from inside the session also ends this client with an error.
	if err == nil || !IsTmuxServerRunning() {
		return nil
	}
	return &commandError{subcommand: "attach-session", stderr: stderr.String(), err: err}
}

// createSession creates a session from the given layout, which must have at least one
// window, and returns it. It removes the session again if a later window fails, so a
// failure never leaves half a session behind.
//
// The first window comes from the new-session command, and the rest from new-window.
// Each window can optionally specify a command to run upon creation.
func createSession(session models.Session) (Session, error) {
	if len(session.Layout.Windows) == 0 {
		return Session{}, errors.New("no windows defined in session layout")
	}

	first := session.Layout.Windows[0]
	output, err := run(buildWindowArgs(true, SessionName(session.Name), first.Name, session.Path, first.Cmd)...)
	if err != nil {
		return Session{}, err
	}

	sessions := parseSessions(output)
	if len(sessions) != 1 {
		return Session{}, fmt.Errorf("unexpected output from tmux new-session: %q", output)
	}
	created := sessions[0]

	for _, w := range session.Layout.Windows[1:] {
		if _, err := run(buildWindowArgs(false, created.ID, w.Name, session.Path, w.Cmd)...); err != nil {
			return Session{}, errors.Join(err, KillSession(created))
		}
	}

	return created, nil
}

// buildWindowArgs constructs tmux command arguments for creating a window.
//
// For the first window it uses new-session, where session is the new session's name, and
// asks tmux to print the result. For later windows it uses new-window, where session is
// the target's $id. If cmd is provided, it wraps it with shell execution to keep the
// window open.
func buildWindowArgs(isFirst bool, session, windowName, dir, cmd string) []string {
	dir = escapeFormat(dir)

	var args []string
	if isFirst {
		args = []string{"new-session", "-d", "-P", "-F", sessionFormat, "-s", session, "-n", windowName, "-c", dir}
	} else {
		args = []string{"new-window", "-t", session, "-n", windowName, "-c", dir}
	}

	if cmd != "" {
		shell := os.Getenv(constants.EnvShell)
		if shell == "" {
			shell = DefaultShell
		}
		cmdStr := cmd + "; exec " + shell
		args = append(args, "--", shell, "-lc", cmdStr)
	}

	return args
}

// escapeFormat makes tmux take s literally where it would expand formats. "##" means a
// plain "#", except that tmux leaves any run of "#" before "[" as it is, because that
// marks a style. Those runs already come through unchanged, so they stay as they are.
func escapeFormat(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '#' {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := i
		for end < len(s) && s[end] == '#' {
			end++
		}
		run := s[i:end]
		b.WriteString(run)
		if end == len(s) || s[end] != '[' {
			b.WriteString(run)
		}
		i = end
	}
	return b.String()
}

// KillSession terminates a session.
func KillSession(session Session) error {
	_, err := run("kill-session", "-t", session.ID)
	return err
}

// KillServer terminates the entire tmux server and all sessions.
//
// This is a destructive operation that will close all tmux sessions.
// Returns an error if the kill operation fails.
func KillServer() error {
	_, err := run("kill-server")
	return err
}

// ReplaceLastSession swaps current, the only session left on the server, for a fresh
// session from the default template, and moves the client to it.
//
// The old session is killed last, because that also ends this process when it runs in
// one of the old session's panes. Until then any failure leaves the old session as it
// was.
func ReplaceLastSession(cfg *models.Config, current Session) error {
	tmpl, found := config.DefaultTemplate(cfg)
	if !found {
		return errors.New("no default template configured")
	}

	// tmux doesn't expand "~" itself, and would open the session in the home directory.
	templatePath := tmpl.Path
	if templatePath == "" {
		templatePath = "~"
	}
	sessionPath, err := utility.ResolvePath(templatePath)
	if err != nil {
		return fmt.Errorf("resolving the path of template %q: %w", tmpl.Name, err)
	}

	session := models.Session{
		Name:   tmpl.Name,
		Path:   sessionPath,
		Layout: models.SessionLayout{Windows: tmpl.Windows},
	}

	// tmux refuses duplicate names, so the old session gives up the name first.
	renamed := current.Name == SessionName(session.Name)
	if renamed {
		if err := renameSession(current, fmt.Sprintf("muxly-replaced-%d", os.Getpid())); err != nil {
			return err
		}
	}

	created, err := createSession(session)
	if err == nil {
		err = SwitchToSession(created)
	}
	if err != nil {
		return errors.Join(err, undoReplace(created, current, renamed))
	}

	return KillSession(current)
}

// undoReplace removes what a failed ReplaceLastSession created, if anything, and gives
// the old session its name back.
func undoReplace(created, old Session, renamed bool) error {
	var errs []error
	if created.ID != "" {
		errs = append(errs, KillSession(created))
	}
	if renamed {
		errs = append(errs, renameSession(old, old.Name))
	}
	return errors.Join(errs...)
}

// renameSession gives a session a new name.
func renameSession(session Session, name string) error {
	_, err := run("rename-session", "-t", session.ID, name)
	return err
}
