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

		var sessionPath string
		if tmpl.Path != "" {
			resolved, err := utility.ResolvePath(tmpl.Path)
			if err != nil {
				return fmt.Errorf("failed to resolve template path: %w", err)
			}
			sessionPath = resolved
		} else {
			builder := selector.NewBuilder(&state.Cfg, state.Verbose)
			entries, err := builder.BuildEntries(0)
			if err != nil {
				return fmt.Errorf("failed to build directory entries: %w", err)
			}

			names := selector.SortedNames(entries, state.Cfg.Settings.TmuxSessionPrefix)
			choiceStr, err := fzf.SelectWithFzf(names)
			if err != nil {
				if errors.Is(err, fzf.ErrCancelled) {
					return nil
				}
				return fmt.Errorf("selecting with fzf failed: %w", err)
			}
			if choiceStr == "" {
				return nil
			}

			selected, exists := entries[choiceStr]
			if !exists {
				return fmt.Errorf("selected entry not found: %s", choiceStr)
			}
			sessionPath = selected.Path
		}

		sessionName := filepath.Base(sessionPath)

		sess := models.Session{
			Name:   sessionName,
			Path:   sessionPath,
			Layout: models.SessionLayout{Windows: tmpl.Windows},
		}
		return tmux.CreateAndSwitchSession(sess)
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
