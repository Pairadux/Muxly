// Package state holds what muxly loads at startup for the commands to share.
package state

import (
	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
)

var (
	// Cfg is the loaded config. CfgLoad says how far to trust it.
	Cfg models.Config
	// CfgLoad is what went wrong loading the config.
	CfgLoad config.LoadReport
	// CfgFilePath is where the config was read from, or where it would be created.
	CfgFilePath string
	// Verbose is set by the --verbose flag.
	Verbose bool
)
