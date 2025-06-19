// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/Tmux-Sessionizer/internal/fzf"
	"github.com/Pairadux/Tmux-Sessionizer/internal/models"
	"github.com/Pairadux/Tmux-Sessionizer/internal/tmux"
	"github.com/Pairadux/Tmux-Sessionizer/internal/utility"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
) // }}}

var (
	cfg         models.Config
	cfgFileFlag string
	cfgFilePath string
	verbose     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "tms [SESSION]",
	Example: "",
	Short:   "A tool for quickly opening tmux sessions",
	Long:    "A tool for quickly opening tmux sessions\n\nBased on ThePrimeagen's Tmux-Sessionator script.",
	Args:    cobra.MaximumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { // {{{
		if isConfigCommand(cmd) {
			return nil
		}

		if err := verifyExternalUtils(); err != nil {
			return err
		}
		if err := validateConfig(); err != nil {
			return err
		}
		warnOnConfigIssues()

		return nil
	}, // }}}
	PreRunE: func(cmd *cobra.Command, args []string) error { // {{{
		if len(args) == 1 {
			// IDEA: Maybe prompt the user and run the command for them
			switch {
			case args[0] == "init":
				return fmt.Errorf("unknown command %q for %q. Did you mean:\n  tms config init?\n", args[0], cmd.Name())
			case args[0] == "edit":
				return fmt.Errorf("unknown command %q for %q. Did you mean:\n  tms config edit?\n", args[0], cmd.Name())
			default:
				return nil
			}
		}
		return nil
	}, // }}}
	RunE: func(cmd *cobra.Command, args []string) error {
		if verbose {
			fmt.Printf("scan_dirs: %v\n", cfg.ScanDirs)
			fmt.Printf("entry_dirs: %v\n", cfg.EntryDirs)
			fmt.Printf("ignore_dirs: %v\n", cfg.IgnoreDirs)
			fmt.Printf("fallback_session: %v\n", cfg.FallbackSession)
			fmt.Printf("tmux_base: %v\n", cfg.TmuxBase)
			fmt.Printf("default_depth: %v\n", cfg.DefaultDepth)
			fmt.Printf("session_layout: %v\n", cfg.SessionLayout)
		}

		flagDepth, _ := cmd.Flags().GetInt("depth")
		entries, err := buildDirectoryEntries(flagDepth)
		cobra.CheckErr(err)

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			names := make([]string, 0, len(entries))
			for name := range entries {
				names = append(names, name)
			}

			slices.SortFunc(names, func(a, b string) int {
				isTmuxA := strings.HasPrefix(a, cfg.TmuxSessionPrefix)
				isTmuxB := strings.HasPrefix(b, cfg.TmuxSessionPrefix)
				if isTmuxA && !isTmuxB {
					return -1
				}
				if !isTmuxA && isTmuxB {
					return 1
				}
				return strings.Compare(a, b)
			})

			choiceStr, err = fzf.SelectWithFzf(names)
			if err != nil {
				if err.Error() == "user cancelled" {
					return nil
				}
				cobra.CheckErr(err)
			}

			if choiceStr == "" {
				return nil
			}
		}

		sessionName := choiceStr
		// TODO: simplify to cutprefix
		if strings.HasPrefix(choiceStr, cfg.TmuxSessionPrefix) {
			sessionName = strings.TrimPrefix(choiceStr, cfg.TmuxSessionPrefix)
		}

		selectedPath, exists := entries[choiceStr]
		if !exists && args[0] == "" {
			return fmt.Errorf("The name must match an existing directory entry: %s", choiceStr)
		}

		// TODO: this is a bit involved, but I want to retrieve a session layout from a .tms file in the directory of the session to be created, if present
		// This would enable dynamic session layouts based on user preference/setup

		if err := tmux.CreateAndSwitchSession(&cfg, sessionName, selectedPath, cfg.SessionLayout); err != nil {
			return fmt.Errorf("Failed to switch session: %w", err)
		}

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() { // {{{
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
} // }}}

func init() { // {{{
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFileFlag, "config", "", "config file (default $XDG_CONFIG_HOME/tms/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().IntP("depth", "d", 0, "Maximum traversal depth")
} // }}}

// initConfig reads in config file and ENV variables if set.
func initConfig() { // {{{
	if cfgFileFlag != "" {
		cfgFilePath = cfgFileFlag
		viper.SetConfigFile(cfgFilePath)
	} else {
		var configDir string

		xdg_config_home := os.Getenv("XDG_CONFIG_HOME")
		if xdg_config_home != "" {
			configDir = xdg_config_home
		} else {
			var err error
			configDir, err = os.UserConfigDir()
			cobra.CheckErr(err)
		}

		cfgDir := filepath.Join(configDir, "tms")
		viper.AddConfigPath(cfgDir)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
		cfgFilePath = filepath.Join(cfgDir, "config.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

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
		// FIXME: try unmarshalling 1 key at a time
		fmt.Fprintf(os.Stderr, "Issue unmarshalling config file: %v\n", err)
	}
} // }}}

// buildDirectoryEntries creates a map of display names to directory paths by
// processing scan_dirs and entry_dirs from the configuration. It handles
// directory scanning at specified depths, filters out ignored directories,
// excludes the current tmux session, and marks existing tmux sessions with
// a "[TMUX]" prefix.
//
// The flagDepth parameter can override the scanning depth for scan_dirs.
// Returns a map where keys are display names and values are resolved paths
// or session names for existing tmux sessions.
func buildDirectoryEntries(flagDepth int) (map[string]string, error) {
	entries := make(map[string]string)
	existingSessions := tmux.GetTmuxSessionSet()
	currentSession := tmux.GetCurrentTmuxSession()

	ignoreSet := make(map[string]struct{})
	for _, dir := range cfg.IgnoreDirs {
		resolved, err := utility.ResolvePath(dir)
		if err == nil {
			ignoreSet[resolved] = struct{}{}
		}
	}

	addEntry := func(path, prefix string) error {
		resolved, err := utility.ResolvePath(path)
		if err != nil {
			return err
		}

		if _, ignored := ignoreSet[resolved]; ignored {
			return nil
		}

		name := filepath.Base(resolved)
		if name == currentSession {
			return nil
		}

		displayName := name
		if prefix != "" {
			displayName = prefix + "/" + name
		}

		if existingSessions[name] {
			displayName = cfg.TmuxSessionPrefix + name
		}

		entries[displayName] = resolved
		return nil
	}

	// TODO: try to make scandir traversal more effecient
	// Maybe make it resolve paths concurrently
	for _, scanDir := range cfg.ScanDirs {
		prefix := scanDir.Alias
		if err := processScanDir(scanDir, flagDepth, prefix, addEntry); err != nil {
			return nil, err
		}
	}

	for _, entryDir := range cfg.EntryDirs {
		if err := addEntry(entryDir, ""); err != nil {
			return nil, err
		}
	}

	for sessionName := range existingSessions {
		if sessionName == currentSession {
			continue
		}
		displayName := cfg.TmuxSessionPrefix + sessionName
		if _, exists := entries[displayName]; !exists {
			entries[displayName] = sessionName
		}
	}

	return entries, nil
}

// processScanDir processes a ScanDir struct, using the struct's depth
// and the existing depth priority logic from the ScanDir.GetDepth method.
func processScanDir(scanDir models.ScanDir, flagDepth int, prefix string, addEntry func(string, string) error) error {
	defaultDepth := cfg.DefaultDepth
	effectiveDepth := scanDir.GetDepth(flagDepth, defaultDepth)

	resolved, err := utility.ResolvePath(scanDir.Path)
	if err != nil {
		return err
	}

	subDirs, err := utility.GetSubDirs(effectiveDepth, resolved)
	if err != nil {
		return err
	}

	for _, subDir := range subDirs {
		if err := addEntry(subDir, prefix); err != nil {
			return err
		}
	}

	return nil
}

// isConfigCommand checks if the given command or any of its parent commands
// is "config". This is used to skip config validation for commands like
// "tms config init" or "tms config edit", which are intended to manage or
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
// It checks for the presence of a config file and verifies that at least one
// directory is configured for scanning (either scan_dirs or entry_dirs).
// Returns an error with helpful instructions if validation fails.
func validateConfig() error {
	if viper.ConfigFileUsed() == "" {
		return fmt.Errorf("no config file found\nRun 'tms config init' to create one, or use --config to specify a path\n")
	}

	if len(cfg.ScanDirs) == 0 && len(cfg.EntryDirs) == 0 {
		return fmt.Errorf("no directories configured for scanning")
	}

	if len(cfg.SessionLayout.Windows) == 0 {
		return fmt.Errorf("session_layout must have at least one window")
	}

	return nil
}

func warnOnConfigIssues() {
	if cfg.Editor == "" {
		fmt.Fprintln(os.Stderr, "Warning: editor not set, defaulting to 'vi'")
	}

	if cfg.FallbackSession.Name == "" {
		fmt.Fprintln(os.Stderr, "fallback_session.name is missing, defaulting to 'Default'")
	}

	if cfg.FallbackSession.Path == "" {
		fmt.Fprintln(os.Stderr, "fallback_session.path is missing, defaulting to '~/'")
	}

	if len(cfg.FallbackSession.Layout.Windows) == 0 {
		fmt.Fprintln(os.Stderr, "fallback_session.layout.windows is empty, using default layout")
	}
}

func verifyExternalUtils() error {
	var missing []string

	if _, err := exec.LookPath("tmux"); err != nil {
		missing = append(missing, "tmux")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		missing = append(missing, "fzf")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}

	return nil
}
