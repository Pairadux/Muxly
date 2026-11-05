package cmd

import (
	"fmt"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/selector"
	"github.com/Pairadux/muxly/internal/session"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/tmux"
)

// openName opens the directory with the given label, or else the session with that name.
func openName(name string, dirs []selector.Directory, sessions []tmux.Session) error {
	for _, dir := range dirs {
		if dir.Label == name {
			return openDirectory(dir, directoryLayout(dir), sessions)
		}
	}
	created := models.Session{Name: name, Layout: directoryLayout(selector.Directory{})}
	if err := tmux.CreateAndSwitchSession(created); err != nil {
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
