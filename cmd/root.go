// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/tms/internal/fzf"
	"github.com/Pairadux/tms/internal/models"
	"github.com/Pairadux/tms/internal/tmux"
	"github.com/Pairadux/tms/internal/utility"

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
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if err := utility.VerifyExternalUtils(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if verbose {
			fmt.Printf("scan_dirs: %v\n", cfg.ScanDirs)
			fmt.Printf("entry_dirs: %v\n", cfg.EntryDirs)
			fmt.Printf("ignore_dirs: %v\n", cfg.IgnoreDirs)
			fmt.Printf("fallback_session: %v\n", cfg.FallbackSession)
			fmt.Printf("tmux_base: %v\n", cfg.TmuxBase)
			fmt.Printf("default_depth: %v\n", cfg.DefaultDepth)
			fmt.Printf("session_layout: %v\n", cfg.SessionLayout)
		}

		if err := utility.ValidateConfig(&cfg); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
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
					os.Exit(0)
				}
				cobra.CheckErr(err)
			}

			if choiceStr == "" {
				os.Exit(0)
			}
		}

		sessionName := choiceStr
		if strings.HasPrefix(choiceStr, cfg.TmuxSessionPrefix) {
			sessionName = strings.TrimPrefix(choiceStr, cfg.TmuxSessionPrefix)
		}

		selectedPath, exists := entries[choiceStr]
		if !exists && args[0] == "" {
			fmt.Fprintf(os.Stderr, "The name must match an existing directory entry: %s\n", choiceStr)
			os.Exit(1)
		}

		// TODO: this is a bit involved, but I want to retrieve a session layout from a .tms file in the directory of the session to be created, if present
		// This would enable dynamic session layouts based on user preference/setup

		if err := tmux.CreateAndSwitchSession(&cfg, sessionName, selectedPath, cfg.SessionLayout); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to switch session: %v\n", err)
			os.Exit(1)
		}
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

	addEntry := func(path string) error {
		resolved, err := utility.ResolvePath(path)
		if err != nil {
			return err
		}

		name := filepath.Base(resolved)
		if name == currentSession {
			return nil
		}

		// TODO: try to find a more effecient way to ignore directories
		// Maybe expand scan_dirs to hold relevant ignore_dirs so the ignore_dirs are not searched for in every single scan_dir
		// Might could also create a seperate ignore_regex to ignore paths based on a regex match
		ignoreDirs := cfg.IgnoreDirs
		for _, ignoreDir := range ignoreDirs {
			ignoredResolved, err := utility.ResolvePath(ignoreDir)
			if err != nil {
				if name == ignoreDir {
					return nil
				}
				continue
			}
			if resolved == ignoredResolved {
				return nil
			}
		}

		displayName := name
		if existingSessions[name] {
			displayName = cfg.TmuxSessionPrefix + name
		}

		entries[displayName] = resolved
		return nil
	}

	// TODO: try to make scandir traversal more effecient
	// Maybe make it resolve paths concurrently
	for _, scanDir := range cfg.ScanDirs {
		if err := processScanDir(scanDir, flagDepth, addEntry); err != nil {
			return nil, err
		}
	}

	for _, entryDir := range cfg.EntryDirs {
		if err := addEntry(entryDir); err != nil {
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
func processScanDir(scanDir models.ScanDir, flagDepth int, addEntry func(string) error) error {
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
		if err := addEntry(subDir); err != nil {
			return err
		}
	}

	return nil
}
