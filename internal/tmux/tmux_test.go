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
			expected:   []string{"new-window", "-t", "$1", "-n", "editor", "-c", "/home/user/project"},
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
			expected:   []string{"new-window", "-t", "$1", "-n", "server", "-c", "/home/user/code", "--", "/bin/zsh", "-lc", "npm run dev; exec /bin/zsh"},
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

func TestNewSessionName(t *testing.T) {
	tests := map[string]string{
		"plain":     "plain",
		"foo.bar":   "foo_bar",
		"a:b":       "a_b",
		".conf/app": "_conf/app",
		"x.y/src":   "x_y/src",
		"$dollar":   "$dollar",
		"=eq":       "=eq",
	}

	for name, want := range tests {
		if got := newSessionName(name); got != want {
			t.Errorf("newSessionName(%q) = %q, want %q", name, got, want)
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
