package config

import (
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/spf13/cobra"
)

// Cmd represents the config command
var Cmd = &cobra.Command{
	Use:   "config",
	Short: "Manage application configuration",
	Long:  "Manage the applications configuration\n\nUse 'config init' to create a new config file and 'config edit' to modify it.",
	// config subcommands manipulate the config file itself and never need
	// tmux/fzf, so skip the external-utility check for the entire subtree.
	Annotations: map[string]string{
		constants.AnnotationSkipUtilsCheck: "true",
	},
}
