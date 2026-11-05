package cmd

import (
	"errors"
	"fmt"

	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/tmux"

	"github.com/spf13/cobra"
)

var skipEnter bool

// switchCmd represents the switch command
var switchCmd = &cobra.Command{
	Use:   "switch [SESSION]",
	Short: "Switch to an active session",
	Long: `Switch to an active session

Displays a fzf picker list of active sessions.
If no other sessions found, exit.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		current, inside := tmux.CurrentSession()
		if !inside {
			return fmt.Errorf("not in Tmux, use 'muxly' to get started")
		}

		sessions, err := tmux.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			others := tmux.Without(sessions, current)

			if len(others) == 0 {
				fmt.Println("No other sessions available. Press 'Enter' to continue...")
				_, err := fmt.Scanln()
				if err != nil {
					return fmt.Errorf("input closed: %w", err)
				}
				return nil
			}

			choiceStr, err = fzf.SelectWithFzf(tmux.Names(others))
			if err != nil {
				if errors.Is(err, fzf.ErrCancelled) {
					return nil
				}
				return fmt.Errorf("selecting with fzf failed: %w", err)
			}

			if choiceStr == "" {
				return nil
			}
		}

		target, found := tmux.FindSession(sessions, choiceStr)
		if !found {
			return fmt.Errorf("no session is named %q\nRun 'muxly switch' to pick one", choiceStr)
		}

		if err := tmux.SwitchToSession(target); err != nil {
			return fmt.Errorf("failed to switch session: %w", err)
		}

		return nil
	}}

func init() {
	rootCmd.AddCommand(switchCmd)
	rootCmd.Flags().BoolVarP(&skipEnter, "skip-enter", "s", false, "Skip the 'press Enter' confirmation prompt when no other sessions found")
}
