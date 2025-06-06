// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize config file",
	Long: `Initialize config file

Creates a config file at the specified location (default location if no argument passed) if no config file exists.
Otherwise, the current config file is overwritten.
The flags provided are used to overwrite those values in the config file.
Any flags that are omitted will be assigned the default values shown.`,
	Run: func(cmd *cobra.Command, args []string) {
		if paths, _ := cmd.Flags().GetStringArray("default_paths"); len(paths) > 0 {
			viper.SetDefault("default_paths", paths)
		}
		viper.SetDefault("example_default", "test")

		parent := filepath.Dir(cfgFilePath)
		_ = os.MkdirAll(parent, 0o755)

		if err := viper.WriteConfigAs(cfgFilePath); err != nil {
			if _, ok := err.(viper.ConfigFileAlreadyExistsError); ok {
				cobra.CheckErr(viper.WriteConfig())
			} else {
				fmt.Fprintln(os.Stderr, "cannot write config:", err)
				os.Exit(1)
			}
		}

		fmt.Println("Wrote config to", cfgFilePath)
	},
}

func init() {// {{{
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	initCmd.Flags().StringArrayP("default_paths", "d", []string{"~/Dev", "~/.dotfiles"}, "A list of paths that should always be searched.")	
}// }}}

