// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Pairadux/tms/internal/models"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)// }}}

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
		// TODO: move all of this to a manually created config file
		// So that comments and such can be added to explain each option
		fresh := viper.New()
		fresh.SetConfigFile(cfgFilePath)

		if scanDirs, _ := cmd.Flags().GetStringArray("scan_dirs"); len(scanDirs) > 0 {
			fresh.Set("scan_dirs", scanDirs)
		}
		if entryDirs, _ := cmd.Flags().GetStringArray("entry_dirs"); len(entryDirs) > 0 {
			fresh.Set("entry_dirs", entryDirs)
		}
		if ignoreDirs, _ := cmd.Flags().GetStringArray("ignore_dirs"); len(ignoreDirs) > 0 {
			fresh.Set("ignore_dirs", ignoreDirs)
		}
		if defaultSession, _ := cmd.Flags().GetString("default_session"); defaultSession != "" {
			fresh.Set("fallback_session", defaultSession)
		}
		if tmuxBase, _ := cmd.Flags().GetInt("tmux_base"); tmuxBase >= 0 {
			fresh.Set("tmux_base", tmuxBase)
		}
		if defaultDepth, _ := cmd.Flags().GetInt("default_depth"); defaultDepth >= 0 {
			fresh.Set("default_depth", defaultDepth)
		}

		sessionLayout := models.SessionLayout{
			Windows: []models.Window{
				{Name: "edit", Cmd: "nvim"},
				{Name: "term", Cmd: ""},
			},
		}
		fresh.Set("session_layout", sessionLayout)


		parent := filepath.Dir(cfgFilePath)
		_ = os.MkdirAll(parent, 0o755)

		if err := fresh.WriteConfigAs(cfgFilePath); err != nil {
			fmt.Fprintln(os.Stderr, "cannot write config:", err)
			os.Exit(1)
		}
		if verbose {
			fmt.Println("Wrote config to", cfgFilePath)
		}
	},
}

func init() { // {{{
	rootCmd.AddCommand(initCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// initCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// initCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	initCmd.Flags().IntP("tmux_base", "b", 1, "What number your windows start ordering at.")
	initCmd.Flags().IntP("default_depth", "d", 1, "Default depth to scan.")
	initCmd.Flags().StringP("default_session", "D", "Documents", "The name of the default session to fall back to.")
	initCmd.Flags().StringArrayP("scan_dirs", "s", []string{"~/Dev", "~/.dotfiles/dot_config"}, "A list of paths that should always be scanned.\nConcat with :int for depth.")
	initCmd.Flags().StringArrayP("entry_dirs", "e", []string{"~/Documents", "~/Cloud"}, "A list of paths that are entries themselves.")
	initCmd.Flags().StringArrayP("ignore_dirs", "i", []string{"~/Dev/_practice", "~/Dev/CS-Homework"}, "A list of paths that should be removed.")
} // }}}
