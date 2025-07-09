// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"

	"github.com/Pairadux/Tmux-Sessionizer/internal/forms"
	"github.com/Pairadux/Tmux-Sessionizer/internal/models"
	"github.com/Pairadux/Tmux-Sessionizer/internal/tmux"

	"github.com/spf13/cobra"
) // }}}

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a session",
	Long: `Create a session

An interactive prompt for creating a session.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Needed for a session:
		// Session Name
		// Session Path
		// Session Layout
		//
		// Prompts:
		// if Use Default:
		// Proceed with default session creation
		// else:
		// Enter Session Name
		// Verify the name isn't already in use
		// Path
		// Use home?
		// Use CWD?
		// Enter Path
		// Layout
		// Use default layout?
		// Enter window 1 name
		// Enter window 1 cmd
		// Repeat for however many windows
		// Present user with a finalized session and ask for confifrmation before creating and entering session

		// FORM VARS
		var (
			useFallback   bool
			confirmCreate bool
			sessionName   string
			path          string
			windowsStr    string
		)

		form := forms.CreateForm(&useFallback, &confirmCreate, &sessionName, &path, &windowsStr)
		if err := form.Run(); err != nil {
			return fmt.Errorf("form error: %w", err)
		}

		// layout := parseWindows(windowsStr)
		layout := cfg.SessionLayout

		session := models.Session{
			Name:   sessionName,
			Path:   path,
			Layout: layout,
		}

		if confirmCreate {
			if useFallback {
				if err := tmux.CreateAndSwitchToFallbackSession(&cfg); err != nil {
					return fmt.Errorf("Failed to create default session: %w", err)
				}
			} else {
				if err := tmux.CreateAndSwitchSession(&cfg, session); err != nil {
					return fmt.Errorf("failed to create session: %w", err)
				}
			}
		} else {
			return nil
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}

// parseWindows parses a comma-delimited input string where each value is a name:cmd pair.
//
// It converts each name:cmd pair into Window structs for the session layout.
// If no colon is found in a part, the entire part is treated as the window name with no command.
// Returns a SessionLayout with at least one window, defaulting to "main" if input is empty.
func parseWindows(input string) models.SessionLayout {
	// TODO: Implement parseWindows function - currently returns empty layout
	return models.SessionLayout{}
}
