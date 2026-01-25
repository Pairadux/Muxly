package cmd


import (
	"errors"
	"fmt"

	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/tmux"

	"github.com/spf13/cobra"
)

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch [SESSION]",
	Short: "Switch to an active session",
	Long: `Switch to an active session

Displays a fzf picker list of active sessions.
If no other sessions found, exit.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		currentSession := tmux.GetCurrentTmuxSession()

		if currentSession == "" {
			return fmt.Errorf("Not in Tmux, use 'muxly' to get started.")
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			sessions := tmux.GetSessionsExceptCurrent(currentSession)

			if len(sessions) == 0 {
				fmt.Println("No other sessions available. Use 'muxly' to start a new session.")
				return nil
			}

			var err error
			choiceStr, err = fzf.SelectWithFzf(sessions)
			if err != nil {
				if err.Error() == "user cancelled" {
					return nil
				}
				return fmt.Errorf("selecting with fzf failed: %w", err)
			}

			if choiceStr == "" {
				return nil
			}
		}
		sessionName := choiceStr
		if err := tmux.SwitchToExistingSession(&cfg, sessionName); err != nil {
			if errors.Is(err, tmux.ErrGracefulExit) {
				return nil
			}
			return fmt.Errorf("Failed to switch session: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(switchCmd)
}
