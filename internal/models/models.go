// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package models

import "fmt"

type Window struct {
	Name string `mapstructure:"name"`
	Cmd  string `mapstructure:"cmd,omitempty"`
}

type SessionLayout struct {
	Windows []Window `mapstructure:"windows"`
}

type ScanDir struct {
	Path  string `mapstructure:"path"`
	Depth *int   `mapstructure:"depth,omitempty"`
}

// WIP
type Session struct {
	Name string
	Layout SessionLayout
}

// GetDepth returns the depth for this scan directory, with fallback logic
func (s ScanDir) GetDepth(flagDepth, defaultDepth int) int {
	if flagDepth > 0 {
		return flagDepth
	}
	if s.Depth != nil {
		return *s.Depth
	}
	if defaultDepth > 0 {
		return defaultDepth
	}
	return 1
}

// String returns the string representation
func (s ScanDir) String() string {
	if s.Depth != nil {
		return fmt.Sprintf("%s:%d", s.Path, *s.Depth)
	}
	return s.Path
}

// Config represents the full configuration structure
type Config struct {
	ScanDirs        []ScanDir     `mapstructure:"scan_dirs"`
	EntryDirs       []string      `mapstructure:"entry_dirs"`
	IgnoreDirs      []string      `mapstructure:"ignore_dirs"`
	FallbackSession string        `mapstructure:"fallback_session"`
	TmuxBase        int           `mapstructure:"tmux_base"`
	DefaultDepth    int           `mapstructure:"default_depth"`
	SessionLayout   SessionLayout `mapstructure:"session_layout"`
}
