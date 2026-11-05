package selector

import (
	"slices"
	"testing"

	"github.com/Pairadux/muxly/internal/tmux"
)

const prefix = "[TMUX] "

func TestEntries(t *testing.T) {
	foo := tmux.Session{ID: "$1", Name: "foo", Path: "/dev/foo"}

	tests := []struct {
		name     string
		dirs     []Directory
		sessions []tmux.Session
		current  tmux.Session
		want     []string
	}{
		{name: "nothing"},
		{
			name: "directories sort ignoring case",
			dirs: []Directory{{Path: "/b", Label: "Beta"}, {Path: "/a", Label: "alpha"}, {Path: "/g", Label: "Gamma"}},
			want: []string{"alpha", "Beta", "Gamma"},
		},
		{
			name:     "sessions come first, and don't rely on the prefix to",
			dirs:     []Directory{{Path: "/a", Label: "apple"}},
			sessions: []tmux.Session{{ID: "$1", Name: "zebra", Path: "/z"}, {ID: "$2", Name: "kiwi", Path: "/k"}},
			want:     []string{prefix + "kiwi", prefix + "zebra", "apple"},
		},
		{
			name:     "a session replaces its directory's row",
			dirs:     []Directory{{Path: "/dev/foo", Label: "foo"}, {Path: "/dev/bar", Label: "bar"}},
			sessions: []tmux.Session{foo},
			want:     []string{prefix + "foo", "bar"},
		},
		{
			name:     "a session with the same name in another folder leaves the directory alone",
			dirs:     []Directory{{Path: "/dev/foo", Label: "foo"}},
			sessions: []tmux.Session{{ID: "$1", Name: "foo", Path: "/elsewhere"}},
			want:     []string{prefix + "foo", "foo"},
		},
		{
			name:     "a session with no path matches nothing",
			dirs:     []Directory{{Path: "/dev/foo", Label: "foo"}},
			sessions: []tmux.Session{{ID: "$1", Name: "scratch"}},
			want:     []string{prefix + "scratch", "foo"},
		},
		{
			name:     "a name that doesn't resemble the directory shows its label",
			dirs:     []Directory{{Path: "/dev/backend", Label: "Dev/backend"}},
			sessions: []tmux.Session{{ID: "$1", Name: "api", Path: "/dev/backend"}},
			want:     []string{prefix + "api (Dev/backend)"},
		},
		{
			name: "names muxly would use don't repeat the label",
			dirs: []Directory{
				{Path: "/a", Label: "MyProject"},
				{Path: "/b/.config", Label: ".config"},
				{Path: "/c/src", Label: "Dev/src"},
				{Path: "/d/old", Label: "proj/old"},
			},
			sessions: []tmux.Session{
				{ID: "$1", Name: "myproject", Path: "/a"},
				{ID: "$2", Name: "config", Path: "/b/.config"},
				{ID: "$3", Name: "dev-src", Path: "/c/src"},
				{ID: "$4", Name: "old", Path: "/d/old"},
			},
			want: []string{prefix + "config", prefix + "dev-src", prefix + "myproject", prefix + "old"},
		},
		{
			name:     "two sessions in one directory both get a row",
			dirs:     []Directory{{Path: "/dev/foo", Label: "foo"}},
			sessions: []tmux.Session{foo, {ID: "$2", Name: "other", Path: "/dev/foo"}},
			want:     []string{prefix + "foo", prefix + "other (foo)"},
		},
		{
			name:     "the current session and its directory are both hidden",
			dirs:     []Directory{{Path: "/dev/foo", Label: "foo"}, {Path: "/dev/bar", Label: "bar"}},
			sessions: []tmux.Session{foo, {ID: "$2", Name: "other", Path: "/elsewhere"}},
			current:  foo,
			want:     []string{prefix + "other", "bar"},
		},
		{
			name:     "another session in the current session's directory is still offered",
			dirs:     []Directory{{Path: "/dev/foo", Label: "foo"}},
			sessions: []tmux.Session{foo, {ID: "$2", Name: "other", Path: "/dev/foo"}},
			current:  foo,
			want:     []string{prefix + "other (foo)"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Labels(Entries(tt.dirs, tt.sessions, tt.current, prefix))
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("rows = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEntriesPointAtWhatTheyOpen(t *testing.T) {
	session := tmux.Session{ID: "$1", Name: "foo", Path: "/dev/foo"}
	dirs := []Directory{{Path: "/dev/foo", Label: "foo"}, {Path: "/dev/bar", Label: "bar", Template: "minimal"}}

	entries := Entries(dirs, []tmux.Session{session}, tmux.Session{}, prefix)

	if len(entries) != 2 {
		t.Fatalf("got %d rows, want 2", len(entries))
	}
	if entries[0].Session == nil || entries[0].Session.ID != "$1" || entries[0].Directory != nil {
		t.Errorf("the session's row = %+v", entries[0])
	}
	if entries[1].Directory == nil || entries[1].Directory.Template != "minimal" || entries[1].Session != nil {
		t.Errorf("the directory's row = %+v", entries[1])
	}
}

// With an empty prefix, a session's row can look exactly like a directory's row. They're
// still separate rows, and the index from the picker tells them apart.
func TestEntriesWithNoPrefixStayDistinct(t *testing.T) {
	dirs := []Directory{{Path: "/dev/foo", Label: "foo"}}
	sessions := []tmux.Session{{ID: "$1", Name: "foo", Path: "/elsewhere"}}

	entries := Entries(dirs, sessions, tmux.Session{}, "")

	if got, want := Labels(entries), []string{"foo", "foo"}; !slices.Equal(got, want) {
		t.Fatalf("rows = %q, want %q", got, want)
	}
	if entries[0].Session == nil || entries[1].Directory == nil {
		t.Errorf("the session should come first: %+v", entries)
	}
}
