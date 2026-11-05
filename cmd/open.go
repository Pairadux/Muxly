package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/selector"
	"github.com/Pairadux/muxly/internal/session"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/tmux"
	"github.com/Pairadux/muxly/internal/utility"
)

// printWarnings reports what's worth knowing but doesn't stop anything. Printing while the
// picker is open would draw over it, so callers do this before it opens or after it closes.
func printWarnings(warnings []string) {
	for _, warning := range warnings {
		fmt.Fprintln(os.Stderr, "Warning:", warning)
	}
}

// openArgument opens what an argument to muxly means. The first of these that fits wins:
// an explicit path, the exact name of a running session, a configured directory, anything
// with a slash as a relative path, and otherwise the name of a new session.
func openArgument(arg string, dirs []selector.Directory, sessions []tmux.Session) error {
	if utility.IsExplicitPath(arg) {
		return openPath(arg, dirs, sessions)
	}

	if existing, found := tmux.FindSession(sessions, arg); found {
		if err := tmux.SwitchToSession(existing); err != nil {
			return fmt.Errorf("failed to switch session: %w", err)
		}
		return nil
	}

	matches := selector.Find(dirs, arg)
	if len(matches) == 1 {
		return openDirectory(matches[0], directoryLayout(matches[0]), sessions)
	}
	if len(matches) > 1 {
		labels := make([]string, len(matches))
		for i, match := range matches {
			labels[i] = match.Label
		}
		return fmt.Errorf("%q could mean any of these directories: %s\nName one of them, or run 'muxly' to pick", arg, strings.Join(labels, ", "))
	}

	if strings.ContainsRune(arg, filepath.Separator) {
		return openPath(arg, dirs, sessions)
	}
	return openNamedSession(arg)
}

// openPath opens a directory given as a path on the command line, which doesn't have to be
// in the config. If it is, the config's alias and template apply.
func openPath(arg string, dirs []selector.Directory, sessions []tmux.Session) error {
	path, err := utility.ResolveArgumentPath(arg)
	if err != nil {
		return fmt.Errorf("resolving %s: %w", arg, err)
	}
	dir, configured := selector.FindPath(dirs, path)
	if !configured {
		dir = selector.Directory{Path: path, Label: filepath.Base(path)}
	}
	return openDirectory(dir, directoryLayout(dir), sessions)
}

// openNamedSession opens the session with a name that means nothing else. It's created in
// the current folder from the default template. A .muxly file there isn't used, because
// a name says nothing about wanting to open that folder, unlike "muxly .".
func openNamedSession(name string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("finding the current folder: %w", err)
	}
	session := models.Session{Name: name, Path: cwd, Layout: defaultLayout()}
	if err := tmux.CreateAndSwitchSession(session); err != nil {
		return fmt.Errorf("failed to switch session: %w", err)
	}
	return nil
}

// openDirectory moves to the session running in a directory, creating one from layout if
// there isn't one.
func openDirectory(dir selector.Directory, layout models.SessionLayout, sessions []tmux.Session) error {
	if existing, found := tmux.FindSessionAt(sessions, dir.Path); found {
		if err := tmux.SwitchToSession(existing); err != nil {
			return fmt.Errorf("failed to switch session: %w", err)
		}
		return nil
	}

	// tmux would open a session in the home directory rather than refuse.
	if err := utility.RequireDirectory(dir.Path); err != nil {
		if dir.Missing {
			return fmt.Errorf("%w\nFix its path in your config, or run 'muxly config remove entry %s'", err, dir.Path)
		}
		return err
	}

	created := models.Session{
		Name:   tmux.FreeSessionName(sessions, dir.Label, dir.Path),
		Path:   dir.Path,
		Layout: layout,
	}
	if err := tmux.CreateAndSwitchSession(created); err != nil {
		return fmt.Errorf("failed to switch session: %w", err)
	}
	return nil
}

// directoryLayout returns the layout for a new session in a directory: its .muxly file,
// or else its template, or else the default template.
func directoryLayout(dir selector.Directory) models.SessionLayout {
	if layout := session.LoadMuxlyFile(dir.Path); len(layout.Windows) > 0 {
		return layout
	}
	if dir.Template != "" {
		if tmpl, found := config.FindTemplateByName(&state.Cfg, dir.Template); found {
			return models.SessionLayout{Windows: tmpl.Windows}
		}
	}
	return defaultLayout()
}

// defaultLayout returns the default template's layout.
func defaultLayout() models.SessionLayout {
	dflt, found := config.DefaultTemplate(&state.Cfg)
	if !found {
		return models.SessionLayout{}
	}
	return models.SessionLayout{Windows: dflt.Windows}
}
