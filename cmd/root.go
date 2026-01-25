package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/checks"
	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/selector"
	"github.com/Pairadux/muxly/internal/session"
	"github.com/Pairadux/muxly/internal/tmux"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfg         models.Config
	cfgFileFlag string
	cfgFilePath string
	verbose     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "muxly [SESSION]",
	Example: "",
	Short:   "A tool for quickly opening tmux sessions",
	Long:    "A tool for quickly opening tmux sessions\n\nBased on ThePrimeagen's tmux-sessionizer script.",
	Args:    cobra.MaximumNArgs(1),
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if isConfigCommand(cmd) {
			return nil
		}

		if err := checks.VerifyExternalUtils(); err != nil {
			return err
		}
		if err := validateConfig(); err != nil {
			return err
		}
		warnOnConfigIssues()

		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			switch args[0] {
			case "init":
				return fmt.Errorf("unknown command %q for %q. Did you mean:\n  muxly config init?\n", args[0], cmd.Name())
			case "edit":
				return fmt.Errorf("unknown command %q for %q. Did you mean:\n  muxly config edit?\n", args[0], cmd.Name())
			default:
				return nil
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if verbose {
			fmt.Printf("scan_dirs: %v\n", cfg.ScanDirs)
			fmt.Printf("entry_dirs: %v\n", cfg.EntryDirs)
			fmt.Printf("ignore_dirs: %v\n", cfg.IgnoreDirs)
			fmt.Printf("fallback_session: %v\n", cfg.FallbackSession)
			fmt.Printf("tmux_base: %v\n", cfg.Settings.TmuxBase)
			fmt.Printf("default_depth: %v\n", cfg.Settings.DefaultDepth)
			fmt.Printf("session_layout: %v\n", cfg.SessionLayout)
		}

		flagDepth, _ := cmd.Flags().GetInt("depth")
		builder := selector.NewBuilder(&cfg, verbose)
		entries, err := builder.BuildEntries(flagDepth)
		if err != nil {
			return fmt.Errorf("failed to build directory entries: %w", err)
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			// Build list of all entry names for fzf selection
			names := make([]string, 0, len(entries))
			for name := range entries {
				names = append(names, name)
			}

			slices.SortFunc(names, func(a, b string) int {
				isTmuxA := strings.HasPrefix(a, cfg.Settings.TmuxSessionPrefix)
				isTmuxB := strings.HasPrefix(b, cfg.Settings.TmuxSessionPrefix)
				if isTmuxA && !isTmuxB {
					return -1
				}
				if !isTmuxA && isTmuxB {
					return 1
				}
				return strings.Compare(strings.ToLower(a), strings.ToLower(b))
			})

			choiceStr, err = fzf.SelectWithFzf(names)
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

		sessionName, _ := strings.CutPrefix(choiceStr, cfg.Settings.TmuxSessionPrefix)

		selectedPath, exists := entries[choiceStr]
		if !exists && len(args) == 0 {
			return fmt.Errorf("the name must match an existing directory entry: %s", choiceStr)
		}

		sessionLayout := session.LoadMuxlyFile(selectedPath)
		if len(sessionLayout.Windows) == 0 {
			sessionLayout = cfg.SessionLayout
		}

		session := models.Session{
			Name:   sessionName,
			Path:   selectedPath,
			Layout: sessionLayout,
		}

		if err := tmux.CreateAndSwitchSession(&cfg, session); err != nil {
			if errors.Is(err, tmux.ErrGracefulExit) {
				return nil
			}
			return fmt.Errorf("Failed to switch session: %w", err)
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
	rootCmd.PersistentFlags().StringVar(&cfgFileFlag, "config", "", "config file (default $XDG_CONFIG_HOME/muxly/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().IntP("depth", "d", 0, "Maximum traversal depth")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFileFlag != "" {
		cfgFilePath = cfgFileFlag
		viper.SetConfigFile(cfgFilePath)
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
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
		cfgFilePath = filepath.Join(cfgDir, "config.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// Bind environment variables for config overrides
	// Allows MUXLY_* environment variables to override config file values
	viper.SetEnvPrefix("MUXLY")
	viper.BindEnv("settings.editor", "MUXLY_EDITOR", "EDITOR") // Support both MUXLY_EDITOR and standard $EDITOR
	viper.BindEnv("settings.default_depth")                    // MUXLY_DEFAULT_DEPTH
	viper.BindEnv("settings.tmux_base")                        // MUXLY_TMUX_BASE
	viper.BindEnv("settings.tmux_session_prefix")              // MUXLY_TMUX_SESSION_PREFIX
	viper.BindEnv("settings.always_kill_on_last_session")      // MUXLY_ALWAYS_KILL_ON_LAST_SESSION

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Config file is corrupted or unreadable:", err)
			os.Exit(1)
		}
	} else if verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse config file: %v\n", err)
		os.Exit(1)
	}

	// Sync cfgFilePath with the actual config file that was loaded
	// This ensures 'muxly config edit' opens the correct file
	if viper.ConfigFileUsed() != "" {
		cfgFilePath = viper.ConfigFileUsed()
	}
}

// isConfigCommand checks if the given command or any of its parent commands
// is "config". This is used to skip config validation for commands like
// "muxly config init" or "muxly config edit", which are intended to manage or
// create the config file.
func isConfigCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "config" {
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
		return fmt.Errorf("no config file found\nRun 'muxly config init' to create one, or use --config to specify a path\n")
	}

	return config.Validate(&cfg)
}

func warnOnConfigIssues() {
	if cfg.Settings.Editor == "" {
		fmt.Fprintf(os.Stderr, "Warning: editor not set, defaulting to '%s'\n", config.DefaultEditor)
	}

	if cfg.FallbackSession.Name == "" {
		fmt.Fprintf(os.Stderr, "fallback_session.name is missing, defaulting to '%s'\n", config.DefaultFallbackSessionName)
	}

	if cfg.FallbackSession.Path == "" {
		fmt.Fprintf(os.Stderr, "fallback_session.path is missing, defaulting to '%s'\n", config.DefaultFallbackSessionPath)
	}

	if len(cfg.FallbackSession.Layout.Windows) == 0 {
		fmt.Fprintln(os.Stderr, "fallback_session.layout.windows is empty, using default layout")
	}
}
