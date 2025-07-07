// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"

	"github.com/Pairadux/Tmux-Sessionizer/internal/forms"
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

		// FIXME
		var (
			useDefault    bool
			confirmCreate bool
			sessionName   string
			pathOption    string
			customPath    string
			windowsStr    string
		)

		form := forms.CreateForm(&useDefault, &confirmCreate, &sessionName, &pathOption, &customPath, &windowsStr)
		if err := form.Run(); err != nil {
			return fmt.Errorf("form error: %w", err)
		}

		if useDefault {
			if err := tmux.CreateAndSwitchToFallbackSession(&cfg); err != nil {
				return fmt.Errorf("Failed to create default session: %w", err)
			}
		} else {
			if confirmCreate {
				if err := tmux.CreateSessionFromInput(&cfg, sessionName, pathOption, customPath, windowsStr); err != nil {
					return fmt.Errorf("failed to create session: %w", err)
				}
			} else {
				return nil
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}
