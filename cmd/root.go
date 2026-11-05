// Package cmd defines muxly's command line: the root picker and the kill, switch,
// create and doctor commands.
package cmd

import (
	"errors"
	"fmt"
	"os"

	cliconfig "github.com/Pairadux/muxly/cmd/config"
	"github.com/Pairadux/muxly/internal/checks"
	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/selector"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/tmux"

	"github.com/spf13/cobra"
)

var cfgFileFlag string

// Version is set at build time via ldflags
var Version = "dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "muxly [SESSION | DIRECTORY]",
	Version: Version,
	Example: `  muxly                pick from running sessions and configured directories
  muxly .              open the current folder
  muxly ~/Dev/project  open that directory, configured or not
  muxly myproject      open a configured directory or running session by name
  muxly scratch        anything else opens a new session with that name here`,
	Short: "A tool for quickly opening tmux sessions",
	Long: `A tool for quickly opening tmux sessions

With no argument, muxly shows a picker of running sessions and configured directories.

With an argument, the first of these that fits is what opens:
  1. A path, if it's ".", "..", or starts with "/", "./", "../" or "~/"
  2. The running session with exactly that name
  3. A configured directory, by its label or folder name, ignoring case
  4. A path from the current folder, if it contains a "/"
  5. Otherwise, a new session with that name, in the current folder, from the default
     template

A directory that already has a session goes to that session.

Based on ThePrimeagen's tmux-sessionizer script.`,
	Args: cobra.MaximumNArgs(1),
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Flags and arguments are parsed by now, so any failure from here on isn't a
		// usage mistake and shouldn't print usage. Earlier failures still do.
		cmd.SilenceUsage = true

		if isCompletionCommand(cmd) {
			return nil
		}

		if err := loadConfig(); err != nil {
			return err
		}

		if !hasAnnotation(cmd, constants.AnnotationSkipUtilsCheck) {
			if err := checks.VerifyExternalUtils(); err != nil {
				return err
			}
		}
		if !hasAnnotation(cmd, constants.AnnotationSkipConfigCheck) {
			// The picker hasn't opened yet, so this can't draw over it.
			printWarnings(state.CfgLoad.Warnings)
			if err := validateConfig(); err != nil {
				return err
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if state.Verbose {
			fmt.Printf("scan_dirs: %v\n", state.Cfg.ScanDirs)
			fmt.Printf("entry_dirs: %v\n", state.Cfg.EntryDirs)
			fmt.Printf("ignore_dirs: %v\n", state.Cfg.IgnoreDirs)
			fmt.Printf("templates: %v\n", state.Cfg.Templates)
			fmt.Printf("default_depth: %v\n", state.Cfg.Settings.DefaultDepth)
		}

		flagDepth, _ := cmd.Flags().GetInt("depth")
		if cmd.Flags().Changed("depth") && flagDepth < 1 {
			return fmt.Errorf("--depth must be at least 1, got %d", flagDepth)
		}

		sessions, err := tmux.ListSessions()
		if err != nil {
			return fmt.Errorf("failed to list sessions: %w", err)
		}
		dirs, warnings := selector.Directories(&state.Cfg, flagDepth)

		if len(args) == 1 && args[0] != "" {
			printWarnings(warnings)
			return openArgument(args[0], dirs, sessions)
		}

		// Outside tmux this is the zero value, which matches no session.
		current, _ := tmux.CurrentSession()
		entries := selector.Entries(dirs, sessions, current, state.Cfg.Settings.TmuxSessionPrefix)
		choice, err := fzf.Pick(selector.Labels(entries))
		printWarnings(warnings)
		if err != nil {
			if errors.Is(err, fzf.ErrCancelled) {
				return nil
			}
			return fmt.Errorf("selecting with fzf failed: %w", err)
		}

		if chosen := entries[choice].Session; chosen != nil {
			if err := tmux.SwitchToSession(*chosen); err != nil {
				return fmt.Errorf("failed to switch session: %w", err)
			}
			return nil
		}

		dir := *entries[choice].Directory
		return openDirectory(dir, directoryLayout(dir), sessions)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(cliconfig.Cmd)
	for _, name := range []string{"add", "edit", "init", "remove"} {
		rootCmd.AddCommand(configCommandPointer(name))
	}
	rootCmd.PersistentFlags().StringVar(&cfgFileFlag, "config", "", "config file (default $XDG_CONFIG_HOME/muxly/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&state.Verbose, "verbose", "V", false, "Enable verbose output")
	rootCmd.Flags().IntP("depth", "d", 0, "Maximum traversal depth")
}

// configCommandPointer stops a command that lives under "muxly config" from being
// taken as a session name when typed at the top level. It's a hidden command rather
// than a check in root, because root would reject the flags and extra arguments of
// something like "muxly add scan ~/Dev --alias dev" before any check ran.
func configCommandPointer(name string) *cobra.Command {
	return &cobra.Command{
		Use:                name,
		Hidden:             true,
		DisableFlagParsing: true,
		Annotations: map[string]string{
			constants.AnnotationSkipUtilsCheck:  "true",
			constants.AnnotationSkipConfigCheck: "true",
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("unknown command %q\nDid you mean 'muxly config %s'?", name, name)
		},
	}
}

// loadConfig finds and loads the config, for the commands to share. What's wrong with it
// goes in state.CfgLoad, since some commands are how you'd fix it. It only fails if there's
// no working out where the config would be.
func loadConfig() error {
	state.CfgFilePath = cfgFileFlag
	if state.CfgFilePath == "" {
		path, err := config.DefaultPath()
		if err != nil {
			return err
		}
		state.CfgFilePath = path
	}

	state.Cfg, state.CfgLoad = config.Load(state.CfgFilePath)
	if state.Verbose && state.CfgLoad.FileErr == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", state.CfgFilePath)
	}
	return nil
}

// isCompletionCommand reports whether cmd is one of cobra's shell completion commands.
// They need no config or tools, but cobra creates them as the program runs, so they
// can't carry annotations like muxly's own commands. That covers "completion" and the
// hidden commands shells call on each Tab press.
func isCompletionCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		switch c.Name() {
		case "completion", cobra.ShellCompRequestCmd, cobra.ShellCompNoDescRequestCmd:
			return true
		}
	}
	return false
}

// hasAnnotation reports whether cmd or any of its ancestors carries the given
// annotation key. Cobra does not propagate annotations to child commands, so
// setting a key on a parent (e.g. the "config" command) applies it to the whole
// subtree, while leaf commands can opt in individually.
func hasAnnotation(cmd *cobra.Command, key string) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if _, ok := c.Annotations[key]; ok {
			return true
		}
	}

	return false
}

// validateConfig returns everything that stops the config being used, along with what to
// do about it.
func validateConfig() error {
	if err := state.CfgLoad.Err(); err != nil {
		return err
	}
	if err := config.Validate(&state.Cfg); err != nil {
		return config.InvalidFileError(state.CfgFilePath, err)
	}
	return nil
}
