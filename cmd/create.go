package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/Pairadux/muxly/internal/forms"
	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/selector"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/tmux"
	"github.com/Pairadux/muxly/internal/utility"

	"github.com/spf13/cobra"
)

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a session from a template",
	Long:  "Create a session from a template\n\nSelect a template, then choose a directory to create the session in.",
	RunE: func(cmd *cobra.Command, args []string) error {
		var selectedIdx int
		form := forms.TemplateSelectForm(state.Cfg.Templates, &selectedIdx)
		if err := form.Run(); err != nil {
			if errors.Is(err, forms.ErrCancelled) {
				return nil
			}
			return fmt.Errorf("template selection failed: %w", err)
		}

		tmpl := state.Cfg.Templates[selectedIdx]

		var dir selector.Directory
		if tmpl.Path != "" {
			resolved, err := utility.ResolvePath(tmpl.Path)
			if err != nil {
				return fmt.Errorf("failed to resolve template path: %w", err)
			}
			dir = selector.Directory{Path: resolved, Label: filepath.Base(resolved)}
		} else {
			dirs, warnings := selector.Directories(&state.Cfg, 0)
			entries := selector.Entries(dirs, nil, tmux.Session{}, "")
			choice, err := fzf.Pick(selector.Labels(entries))
			printWarnings(warnings)
			if err != nil {
				if errors.Is(err, fzf.ErrCancelled) {
					return nil
				}
				return fmt.Errorf("selecting with fzf failed: %w", err)
			}
			dir = *entries[choice].Directory
		}

		sessions, err := tmux.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		return openDirectory(dir, models.SessionLayout{Windows: tmpl.Windows}, sessions)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
