package tmux

import (
	"errors"
	"reflect"
	"testing"
)

func TestBuildWindowArgs(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")

	tests := []struct {
		name       string
		isFirst    bool
		session    string
		windowName string
		dir        string
		cmd        string
		expected   []string
	}{
		{
			name:       "first window no command",
			isFirst:    true,
			session:    "myproject",
			windowName: "main",
			dir:        "/home/user/project",
			cmd:        "",
			expected:   []string{"new-session", "-d", "-P", "-F", sessionFormat, "-s", "myproject", "-n", "main", "-c", "/home/user/project"},
		},
		{
			name:       "subsequent window no command",
			isFirst:    false,
			session:    "$1",
			windowName: "editor",
			dir:        "/home/user/project",
			cmd:        "",
			expected:   []string{"new-window", "-d", "-t", "$1", "-n", "editor", "-c", "/home/user/project"},
		},
		{
			name:       "first window with command",
			isFirst:    true,
			session:    "dev",
			windowName: "vim",
			dir:        "/home/user/code",
			cmd:        "nvim",
			expected:   []string{"new-session", "-d", "-P", "-F", sessionFormat, "-s", "dev", "-n", "vim", "-c", "/home/user/code", "--", "/bin/zsh", "-lc", "nvim; exec /bin/zsh"},
		},
		{
			name:       "subsequent window with command",
			isFirst:    false,
			session:    "$1",
			windowName: "server",
			dir:        "/home/user/code",
			cmd:        "npm run dev",
			expected:   []string{"new-window", "-d", "-t", "$1", "-n", "server", "-c", "/home/user/code", "--", "/bin/zsh", "-lc", "npm run dev; exec /bin/zsh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWindowArgs(tt.isFirst, tt.session, tt.windowName, tt.dir, tt.cmd)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("buildWindowArgs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildWindowArgsDefaultShell(t *testing.T) {
	t.Setenv("SHELL", "")

	got := buildWindowArgs(true, "test", "main", "/tmp", "echo hello")

	expectedShell := DefaultShell
	if got[len(got)-3] != expectedShell {
		t.Errorf("buildWindowArgs should use default shell %q when SHELL not set, got %q", expectedShell, got[len(got)-3])
	}
}

func TestCommandErrorMessage(t *testing.T) {
	cause := errors.New("exit status 1")

	tests := []struct {
		name string
		err  commandError
		want string
	}{
		{
			name: "tmux's own message replaces the exit status",
			err:  commandError{subcommand: "kill-session", stderr: "can't find session: x\n", err: cause},
			want: "tmux kill-session: can't find session: x",
		},
		{
			name: "with nothing from tmux, the cause is all there is",
			err:  commandError{subcommand: "kill-session", stderr: " \n", err: cause},
			want: "tmux kill-session: exit status 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
			if !errors.Is(&tt.err, cause) {
				t.Error("the cause should stay reachable through errors.Is")
			}
		})
	}
}

func TestParseSessions(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   []Session
	}{
		{name: "no sessions", output: "", want: nil},
		{
			name:   "several sessions",
			output: "$1\tfoo\t/a\n$12\tbar baz\t/b c",
			want:   []Session{{"$1", "foo", "/a"}, {"$12", "bar baz", "/b c"}},
		},
		{name: "characters tmux would read as syntax", output: "$1\t$2\t/a", want: []Session{{"$1", "$2", "/a"}}},
		{name: "a session with no path", output: "$1\tfoo\t", want: []Session{{"$1", "foo", ""}}},
		{name: "a trailing slash is cleaned", output: "$1\tfoo\t/a/b/", want: []Session{{"$1", "foo", "/a/b"}}},
		{name: "dot segments are cleaned", output: "$1\tfoo\t/a/../b/./c", want: []Session{{"$1", "foo", "/b/c"}}},
		{name: "a tab in the path", output: "$1\tfoo\t/a\tb", want: []Session{{"$1", "foo", "/a\tb"}}},
		{
			name:   "a newline in the path skips the leftover line",
			output: "$1\tfoo\t/a\nb\n$2\tbar\t/c",
			want:   []Session{{"$1", "foo", "/a"}, {"$2", "bar", "/c"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseSessions(tt.output); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseSessions() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestFindSessionNeedsTheExactName(t *testing.T) {
	sessions := []Session{{"$1", "foobar", ""}, {"$2", "Foo", ""}, {"$3", "x.y", ""}}

	tests := []struct {
		name   string
		want   string
		wantOK bool
	}{
		{name: "foobar", want: "$1", wantOK: true},
		{name: "x.y", want: "$3", wantOK: true},
		{name: "foo"},
		{name: "foob"},
		{name: "x"},
		{name: "$1"},
		{name: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := FindSession(sessions, tt.name)
			if ok != tt.wantOK || got.ID != tt.want {
				t.Errorf("FindSession(%q) = %+v, %v", tt.name, got, ok)
			}
		})
	}
}

func TestWithoutLeavesTheOriginalAlone(t *testing.T) {
	sessions := []Session{{ID: "$1", Name: "a"}, {ID: "$2", Name: "b"}, {ID: "$3", Name: "c"}}

	got := Without(sessions, Session{ID: "$2"})

	if want := []string{"a", "c"}; !reflect.DeepEqual(Names(got), want) {
		t.Errorf("Without() kept %q, want %q", Names(got), want)
	}
	if want := []string{"a", "b", "c"}; !reflect.DeepEqual(Names(sessions), want) {
		t.Errorf("the original became %q", Names(sessions))
	}
}

func TestFindSessionAt(t *testing.T) {
	sessions := []Session{
		{ID: "$10", Name: "newest", Path: "/shared"},
		{ID: "$9", Name: "older", Path: "/shared"},
		{ID: "$12", Name: "newer", Path: "/shared"},
		{ID: "$3", Name: "alone", Path: "/alone"},
		{ID: "$4", Name: "nowhere", Path: ""},
	}

	tests := []struct {
		name      string
		path      string
		want      string
		wantFound bool
	}{
		{name: "the only session there", path: "/alone", want: "alone", wantFound: true},
		{name: "the oldest of several, comparing ids as numbers", path: "/shared", want: "older", wantFound: true},
		{name: "nothing there", path: "/other"},
		{name: "a parent isn't a match", path: "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := FindSessionAt(sessions, tt.path)
			if found != tt.wantFound || got.Name != tt.want {
				t.Errorf("FindSessionAt(%q) = %+v, %v, want %q, %v", tt.path, got, found, tt.want, tt.wantFound)
			}
		})
	}
}

func TestFreeSessionName(t *testing.T) {
	named := func(names ...string) []Session {
		sessions := make([]Session, len(names))
		for i, name := range names {
			sessions[i] = Session{Name: name}
		}
		return sessions
	}

	tests := []struct {
		name     string
		sessions []Session
		label    string
		path     string
		want     string
	}{
		{name: "free", label: "foo", path: "/dev/foo", want: "foo"},
		{name: "made safe", label: ".My Project", path: "/dev/.My Project", want: "my-project"},
		{name: "taken adds the parent", sessions: named("foo"), label: "foo", path: "/dev/foo", want: "dev-foo"},
		{name: "the parent is made safe", sessions: named("foo"), label: "foo", path: "/My.Dev/foo", want: "my_dev-foo"},
		{name: "a longer label adds another parent", sessions: named("dev-src"), label: "Dev/src", path: "/a/Dev/src", want: "dev-dev-src"},
		{name: "that taken too adds a number", sessions: named("foo", "dev-foo"), label: "foo", path: "/dev/foo", want: "foo-2"},
		{name: "numbers go up", sessions: named("foo", "dev-foo", "foo-2", "foo-3"), label: "foo", path: "/dev/foo", want: "foo-4"},
		{name: "no parent to add", sessions: named("foo"), label: "foo", path: "/foo", want: "foo-2"},
		{name: "only the exact name counts as taken", sessions: named("foobar"), label: "foo", path: "/dev/foo", want: "foo"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FreeSessionName(tt.sessions, tt.label, tt.path); got != tt.want {
				t.Errorf("FreeSessionName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSessionName(t *testing.T) {
	tests := map[string]string{
		"plain":         "plain",
		"MyProject":     "myproject",
		"CAFÉ":          "café",
		"日本語":           "日本語",
		"foo.bar":       "foo_bar",
		"a:b":           "a-b",
		"my project":    "my-project",
		"a  b":          "a-b",
		"a--b":          "a-b",
		"a__b":          "a_b",
		"a..b":          "a_b",
		"100%":          "100",
		"a\\b":          "a-b",
		"it's":          "it-s",
		"[brk]":         "brk",
		".config":       "config",
		"..hidden":      "hidden",
		"$dollar":       "dollar",
		"%pct":          "pct",
		"=eq":           "eq",
		"@at":           "at",
		"-dash":         "dash",
		"_under":        "under",
		"~tilde":        "tilde",
		"🚀rocket":       "rocket",
		"é-🎉":           "é",
		"#(touch x)":    "touch-x",
		"#{session_id}": "session_id",
		"a#{b}c":        "a-b-c",
		"new\nline":     "new-line",
		"Dev/src":       "dev-src",
		"proj/unique":   "proj-unique",
		"x.y/src":       "x_y-src",
		".conf/app":     "conf-app",
		"Dev/.config":   "dev-config",
		"/abs/path/":    "abs-path",
		"":              "session",
		"...":           "session",
		"🚀":             "session",
		"/":             "session",
	}

	for label, want := range tests {
		got := SessionName(label)
		if got != want {
			t.Errorf("SessionName(%q) = %q, want %q", label, got, want)
		}
		if again := SessionName(got); again != got {
			t.Errorf("SessionName(%q) = %q changes its own result to %q", label, got, again)
		}
	}
}

func TestNoServer(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "no error", err: nil},
		{name: "another kind of error", err: errors.New("no server running")},
		{name: "stale socket", err: &commandError{stderr: "no server running on /tmp/x\n"}, want: true},
		{name: "no socket", err: &commandError{stderr: "error connecting to /tmp/x (No such file or directory)\n"}, want: true},
		{name: "another tmux failure", err: &commandError{stderr: "protocol version mismatch\n"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := noServer(tt.err); got != tt.want {
				t.Errorf("noServer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeFormat(t *testing.T) {
	tests := map[string]string{
		"plain":        "plain",
		"a#b":          "a##b",
		"a#":           "a##",
		"#":            "##",
		"a##b":         "a####b",
		"a#S":          "a##S",
		"a#{x}b":       "a##{x}b",
		"a#(x)b":       "a##(x)b",
		"a#[b":         "a#[b",
		"a##[b":        "a##[b",
		"a###[b":       "a###[b",
		"a#[#b":        "a#[##b",
		"a#b#[c":       "a##b#[c",
		"#{a}#[b]#(c)": "##{a}#[b]##(c)",
	}

	for input, want := range tests {
		if got := escapeFormat(input); got != want {
			t.Errorf("escapeFormat(%q) = %q, want %q", input, got, want)
		}
	}
}
