// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/Pairadux/tms/internal/fzf"
	"github.com/Pairadux/tms/internal/tmux"
	"github.com/Pairadux/tms/internal/utility"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
) // }}}

var (
	cfgFileFlag string
	cfgFilePath string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tms",
	Short: "A tool for quickly opening tmux sessions",
	Long:  "A tool for quickly opening tmux sessions.\n\nBased on ThePrimeagen's Tmux-Sessionator script.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := tmux.ValidateTmuxAvailable(); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		if err := validateConfig(); err != nil {
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
				isTmuxA := strings.HasPrefix(a, "[TMUX] ")
				isTmuxB := strings.HasPrefix(b, "[TMUX] ")
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
		if strings.HasPrefix(choiceStr, "[TMUX] ") {
			sessionName = strings.TrimPrefix(choiceStr, "[TMUX] ")
		}
		selectedPath, exists := entries[choiceStr]
		if !exists {
			fmt.Fprintf(os.Stderr, "Selected directory not found: %s\n", choiceStr)
			os.Exit(1)
		}
		if err := tmux.TmuxSwitchSession(sessionName, selectedPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to switch session: %v\n", err)
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

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFileFlag, "config", "", "config file (default $XDG_CONFIG_HOME/tms/config.yaml)")

	rootCmd.Flags().IntP("depth", "d", 0, "Maximum traversal depth")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	// rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
} // }}}

// initConfig reads in config file and ENV variables if set.
func initConfig() { // {{{
	if cfgFileFlag != "" {
		// Use config file from the flag.
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
		}
	} else {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
} // }}}

func validateConfig() error {
	if viper.ConfigFileUsed() == "" {
		return fmt.Errorf("no config file found\nRun 'tms init' to create one, or use --config to specify a path")
	}
	if (len(viper.GetStringSlice("scan_dirs")) == 0) && (len(viper.GetStringSlice("entry_dirs")) == 0) {
		return fmt.Errorf("no directories configured for scanning")
	}
	return nil
}

func buildDirectoryEntries(flagDepth int) (map[string]string, error) {
	entries := make(map[string]string)
	existingSessions := tmux.GetTmuxSessions()
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
		ignoreDirs := viper.GetStringSlice("ignore_dirs")
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
			displayName = "[TMUX] " + name
		}
		entries[displayName] = resolved
		return nil
	}
	for _, scanDir := range viper.GetStringSlice("scan_dirs") {
		if err := processScanDir(scanDir, flagDepth, addEntry); err != nil {
			return nil, err
		}
	}
	for _, entryDir := range viper.GetStringSlice("entry_dirs") {
		if err := addEntry(entryDir); err != nil {
			return nil, err
		}
	}
	for sessionName := range existingSessions {
		if sessionName == currentSession {
			continue
		}
		displayName := "[TMUX] " + sessionName
		if _, exists := entries[displayName]; !exists {
			entries[displayName] = sessionName
		}
	}
	return entries, nil
}

func processScanDir(scanDir string, flagDepth int, addEntry func(string) error) error {
	var path string
	var scanDepth int
	if p, depthStr, found := strings.Cut(scanDir, ":"); found {
		if d, err := strconv.Atoi(depthStr); err == nil {
			path = p
			scanDepth = d
		} else {
			path = scanDir
			scanDepth = 0
		}
	} else {
		path = scanDir
		scanDepth = 0
	}
	effectiveDepth := getEffectiveDepth(flagDepth, scanDepth)
	resolved, err := utility.ResolvePath(path)
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

func getEffectiveDepth(scanDepth int, flagDepth int) int {
	if flagDepth > 0 {
		return flagDepth
	}
	if scanDepth > 0 {
		return scanDepth
	}
	defaultDepth := viper.GetInt("default_depth")
	if defaultDepth > 0 {
		return defaultDepth
	}
	return 1
}
