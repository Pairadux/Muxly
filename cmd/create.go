// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"

	"github.com/Pairadux/tms/internal/forms"
	"github.com/Pairadux/tms/internal/models"
	"github.com/Pairadux/tms/internal/tmux"
	"github.com/mitchellh/go-homedir"

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

		var (
			useDefault    bool
			confirmCreate bool
			sessionName   string
			pathOption    string
			customPath    string
			windowsStr    string
			session       models.Session
		)

		forms.CreateForm(&useDefault, &confirmCreate, &sessionName, &pathOption, &customPath, &windowsStr)

		if useDefault {
			if err := tmux.CreateAndSwitchToFallbackSession(&cfg); err != nil {
				return fmt.Errorf("Failed to create default session: %w", err)
			}
		} else {
			if confirmCreate {
				var (
					path string
					err  error
				)
				switch pathOption {
				case "Home":
					path, err = homedir.Dir()
				case "CWD":
					path, err = os.Getwd()
				case "Custom":
					path = customPath
				default:
					return fmt.Errorf("invalid path option %q", pathOption)
				}

				if err != nil {
					return fmt.Errorf("failed to resolve path: %w", err)
				}

				layout := parseWindows(windowsStr)

				session = models.Session{
					Name:   sessionName,
					Path:   path,
					Layout: layout,
				}
			} else {
				return nil
			}
		}

		fmt.Printf("useDefault: %v, sessionName: %s, pathOption %s, customPath %s, windowsStr %s, confirmCreate %v, session %v\n", useDefault, sessionName, pathOption, customPath, windowsStr, confirmCreate, session)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(createCmd)
}

// Accepts a comma delimmited input string where each value is a name:cmd pair
// Parses each name:cmd pair into windows for the session layout
func parseWindows(input string) models.SessionLayout {
	return models.SessionLayout{}
}
