// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Pairadux/tms/internal/utility"
	// "github.com/Pairadux/tms/internal/models"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFileFlag string
	cfgFilePath string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tms",
	Short: "A tool for quickly opening tmux sessions",
	Long:  "A tool for quickly opening tmux sessions.\n\nBased on ThePrimeagen's Tmux-Sessionator script.",
	Run: func(cmd *cobra.Command, args []string) {
		if viper.ConfigFileUsed() == "" {
			fmt.Fprintln(os.Stderr, "No config file found, please generate with `tms init [OPTIONS]\nSee `tms init --help` for additional details.")
			os.Exit(1)
		}

		entries := make(map[string]string)
		var scanDirs []string

		entryDirs := viper.GetStringSlice("entry_dirs")
		maxDepth, err := cmd.Flags().GetInt("depth")
		cobra.CheckErr(err)
		scanDirsPre := viper.GetStringSlice("scan_dirs")

		for _, e := range scanDirsPre {
			resolvedPath, err := utility.ResolvePath(e)
			cobra.CheckErr(err)
			scanDirs = append(scanDirs, resolvedPath)
		}

		for _, scanDir := range scanDirs {
			subDirs, err := utility.GetSubDirs(maxDepth, scanDir)
			cobra.CheckErr(err)
			for _, subDir := range subDirs {
				entries[filepath.Base(subDir)] = subDir
			}
		}

		for _, e := range entryDirs {
			resolvedPath, err := utility.ResolvePath(e)
			cobra.CheckErr(err)
			entries[filepath.Base(resolvedPath)] = resolvedPath
		}

		names := make([]string, 0, len(entries))
		for name := range entries {
			names = append(names, name)
		}
		sort.Strings(names)

		fzf := exec.Command("fzf")
		fzf.Stdin = strings.NewReader(strings.Join(names, "\n"))
		fzf.Stderr = os.Stderr

		choice, err := fzf.Output()
		if err != nil {
			if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 130 {
				os.Exit(0)
			}
			cobra.CheckErr(err)
		}

		choiceStr := strings.TrimSpace(string(choice))
		if choiceStr == "" {
			os.Exit(0)
		}

		selectedPath, exists := entries[choiceStr]
		if !exists {
			fmt.Fprintf(os.Stderr, "Selected directory not found: %s\n", choiceStr)
			os.Exit(1)
		}
		fmt.Printf("%s", selectedPath)

		// command := exec.Command("echo", entries[string(choice)])
		// command.Stdout = os.Stdout
		// command.Run()
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

	rootCmd.Flags().IntP("depth", "d", 1, "Maximum traversal depth")

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
