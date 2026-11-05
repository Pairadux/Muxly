// Package cmd defines muxly's command line: the root picker and the kill, switch,
// create and doctor commands.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cliconfig "github.com/Pairadux/muxly/cmd/config"
	"github.com/Pairadux/muxly/internal/checks"
	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/selector"
	"github.com/Pairadux/muxly/internal/session"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/tmux"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFileFlag string

// Version is set at build time via ldflags
var Version = "dev"

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "muxly [SESSION]",
	Version: Version,
	Example: "",
	Short:   "A tool for quickly opening tmux sessions",
	Long:    "A tool for quickly opening tmux sessions\n\nBased on ThePrimeagen's tmux-sessionizer script.",
	Args:    cobra.MaximumNArgs(1),
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

		if !hasAnnotation(cmd, constants.AnnotationSkipUtilsCheck) {
			if err := checks.VerifyExternalUtils(); err != nil {
				return err
			}
		}
		if !hasAnnotation(cmd, constants.AnnotationSkipConfigCheck) {
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
			fmt.Printf("tmux_base: %v\n", state.Cfg.Settings.TmuxBase)
			fmt.Printf("default_depth: %v\n", state.Cfg.Settings.DefaultDepth)
		}

		flagDepth, _ := cmd.Flags().GetInt("depth")
		builder := selector.NewBuilder(&state.Cfg, state.Verbose)
		entries, err := builder.BuildEntries(flagDepth)
		if err != nil {
			return fmt.Errorf("failed to build directory entries: %w", err)
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			names := selector.SortedNames(entries, state.Cfg.Settings.TmuxSessionPrefix)
			choiceStr, err = fzf.SelectWithFzf(names)
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

		sessionName, _ := strings.CutPrefix(choiceStr, state.Cfg.Settings.TmuxSessionPrefix)

		selected, exists := entries[choiceStr]
		if !exists && len(args) == 0 {
			return fmt.Errorf("the name must match an existing directory entry: %s", choiceStr)
		}

		sessionLayout := session.LoadMuxlyFile(selected.Path)
		if len(sessionLayout.Windows) == 0 && selected.Template != "" {
			if tmpl, found := config.FindTemplateByName(&state.Cfg, selected.Template); found {
				sessionLayout = models.SessionLayout{Windows: tmpl.Windows}
			}
		}
		if len(sessionLayout.Windows) == 0 {
			if dflt, found := config.DefaultTemplate(&state.Cfg); found {
				sessionLayout = models.SessionLayout{Windows: dflt.Windows}
			}
		}

		sess := models.Session{
			Name:   sessionName,
			Path:   selected.Path,
			Layout: sessionLayout,
		}

		if err := tmux.CreateAndSwitchSession(sess); err != nil {
			if errors.Is(err, tmux.ErrGracefulExit) {
				return nil
			}
			return fmt.Errorf("failed to switch session: %w", err)
		}

		return nil
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
	cobra.OnInitialize(initConfig)
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

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFileFlag != "" {
		state.CfgFilePath = cfgFileFlag
		viper.SetConfigFile(state.CfgFilePath)
	} else {
		var configDir string

		xdg_config_home := os.Getenv(constants.EnvXdgConfigHome)
		if xdg_config_home != "" {
			configDir = xdg_config_home
		} else {
			var err error
			configDir, err = os.UserConfigDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "UserConfigDir cannot be found: %v\n", err)
			}
		}

		cfgDir := filepath.Join(configDir, "muxly")
		viper.AddConfigPath(cfgDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
		state.CfgFilePath = filepath.Join(cfgDir, "config.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// Bind environment variables for config overrides
	// Allows MUXLY_* environment variables to override config file values
	// BindEnv only fails when called with no arguments, so its error can't occur here.
	viper.SetEnvPrefix("MUXLY")
	_ = viper.BindEnv("settings.editor", "MUXLY_EDITOR", "EDITOR") // Support both MUXLY_EDITOR and standard $EDITOR
	_ = viper.BindEnv("settings.default_depth")                    // MUXLY_DEFAULT_DEPTH
	_ = viper.BindEnv("settings.tmux_base")                        // MUXLY_TMUX_BASE
	_ = viper.BindEnv("settings.tmux_session_prefix")              // MUXLY_TMUX_SESSION_PREFIX
	_ = viper.BindEnv("settings.always_kill_on_last_session")      // MUXLY_ALWAYS_KILL_ON_LAST_SESSION

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Config file is corrupted or unreadable:", err)
			os.Exit(1)
		}
	} else if state.Verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&state.Cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse config file: %v\n", err)
		os.Exit(1)
	}

	config.ApplyDefaults(&state.Cfg)

	// Sync cfgFilePath with the actual config file that was loaded
	// This ensures 'muxly config edit' opens the correct file
	if viper.ConfigFileUsed() != "" {
		state.CfgFilePath = viper.ConfigFileUsed()
	}
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

// validateConfig ensures that the application configuration is valid and complete.
// It checks for the presence of a config file and validates the config structure.
// Returns an error with helpful instructions if validation fails.
func validateConfig() error {
	if viper.ConfigFileUsed() == "" {
		return fmt.Errorf("no config file found\nRun 'muxly config init' to create one, or use --config to specify a path")
	}

	return config.Validate(&state.Cfg)
}
