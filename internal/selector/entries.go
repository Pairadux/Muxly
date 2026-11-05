package selector

import (
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/tmux"
)

// missingSuffix marks a directory in the picker that can't be opened.
const missingSuffix = " [missing]"

// Entry is one row of the picker. Exactly one of Session and Directory is set.
type Entry struct {
	// Label is what the picker shows.
	Label string
	// Session is the running session this row switches to.
	Session *tmux.Session
	// Directory is the directory this row opens, when it has no running session.
	Directory *Directory
}

// Entries returns the picker's rows: running sessions, then directories with no session,
// each sorted by label ignoring case.
//
// A directory with a running session is represented by that session, however the session
// got its name. The current session is left out, since switching to it does nothing. Its
// directory is too, because opening that would only return to it.
func Entries(dirs []Directory, sessions []tmux.Session, current tmux.Session, sessionPrefix string) []Entry {
	byPath := make(map[string]*Directory, len(dirs))
	for i := range dirs {
		byPath[dirs[i].Path] = &dirs[i]
	}

	var running []Entry
	occupied := make(map[string]bool, len(sessions))
	for _, session := range sessions {
		occupied[session.Path] = true
		if session.ID == current.ID {
			continue
		}
		label := sessionPrefix + session.Name
		if dir, found := byPath[session.Path]; found && !resembles(session.Name, *dir) {
			label += " (" + dir.Label + ")"
		}
		running = append(running, Entry{Label: label, Session: &session})
	}

	var waiting []Entry
	for i := range dirs {
		if occupied[dirs[i].Path] {
			continue
		}
		label := dirs[i].Label
		if dirs[i].Missing {
			label += missingSuffix
		}
		waiting = append(waiting, Entry{Label: label, Directory: &dirs[i]})
	}

	slices.SortFunc(running, compareLabels)
	slices.SortFunc(waiting, compareLabels)
	return append(running, waiting...)
}

// Labels returns the labels of entries, in the same order.
func Labels(entries []Entry) []string {
	labels := make([]string, len(entries))
	for i, entry := range entries {
		labels[i] = entry.Label
	}
	return labels
}

// resembles reports whether a session's name already says which directory it's for, so a
// search for the directory would find it without the directory's label being added.
func resembles(sessionName string, dir Directory) bool {
	name := tmux.SessionName(sessionName)
	return name == tmux.SessionName(dir.Label) || name == tmux.SessionName(filepath.Base(dir.Path))
}

func compareLabels(a, b Entry) int {
	if order := strings.Compare(strings.ToLower(a.Label), strings.ToLower(b.Label)); order != 0 {
		return order
	}
	return strings.Compare(a.Label, b.Label)
}
