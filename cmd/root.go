// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/tms/internal/utility"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFileFlag string

var cfgFilePath string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tms",
	Short: "A tool for quickly opening tmux sessions",
	Long: `A tool for quickly opening tmux sessions

Based on ThePrimeagen's Tmux-Sessionator script.`,
	Run: func(cmd *cobra.Command, args []string) {
		// if viper.GetString("example_default") == "test" {
		// 	fmt.Println("passed")
		// } else {
		// 	fmt.Println("failed")
		// }

		// TODO: replace these hardcoded entries with entries supplied by the entries in the config file and the utility.ResolvePath function
		// {{{
		homeDir, err := os.UserHomeDir()
		cobra.CheckErr(err)
		devDir := filepath.Join(homeDir, "Dev")
		dotfilesDir := filepath.Join(homeDir, ".dotfiles")

		utility.ResolvePath()

		devEntries, err := os.ReadDir(devDir)
		cobra.CheckErr(err)
		dotfilesEntries, err := os.ReadDir(dotfilesDir)
		cobra.CheckErr(err)
		// }}}

		entries := slices.Concat(devEntries, dotfilesEntries)
		cobra.CheckErr(err)

		// TODO: make a function that accepts several directory types and expands them
		dirs := []string{"Documents"}
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, e.Name())
			}
		}

		fzf := exec.Command("fzf")
		fzf.Stdin = strings.NewReader(strings.Join(dirs, "\n"))
		choice, err := fzf.Output()
		cobra.CheckErr(err)

		command := exec.Command("echo", string(choice))
		command.Stdout = os.Stdout
		command.Run()

		depth, err := cmd.Flags().GetInt("depth")
		cobra.CheckErr(err)
		fmt.Printf("Depth: %d\n", depth)
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
