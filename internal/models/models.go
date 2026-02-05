package models

import "fmt"

// StringSet represents a set of strings using a map with empty struct values for memory efficiency
type StringSet map[string]struct{}

type Window struct {
	Name string `mapstructure:"name" yaml:"name"`
	Cmd  string `mapstructure:"cmd,omitempty" yaml:"cmd,omitempty"`
}

type SessionLayout struct {
	Windows []Window `mapstructure:"windows" yaml:"windows"`
}

type SessionTemplate struct {
	Name    string   `mapstructure:"name" yaml:"name"`
	Path    string   `mapstructure:"path,omitempty" yaml:"path,omitempty"`
	Windows []Window `mapstructure:"windows" yaml:"windows"`
}

type ScanDir struct {
	Path  string `mapstructure:"path" yaml:"path"`
	Depth *int   `mapstructure:"depth,omitempty" yaml:"depth,omitempty"`
	Alias string `mapstructure:"alias,omitempty" yaml:"alias,omitempty"`
}

type Session struct {
	Name   string        `mapstructure:"name"`
	Path   string        `mapstructure:"path"`
	Layout SessionLayout `mapstructure:"layout"`
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
	result := s.Path
	if s.Depth != nil {
		result = fmt.Sprintf("%s:%d", s.Path, *s.Depth)
	}
	if s.Alias != "" {
		result = fmt.Sprintf("%s (alias: %s)", result, s.Alias)
	}
	return result
}

// PathInfo holds metadata for a directory path including any alias prefix
type PathInfo struct {
	Path   string
	Prefix string
}

// Settings groups general configuration options
type Settings struct {
	Editor                  string `mapstructure:"editor" yaml:"editor"`
	TmuxBase                int    `mapstructure:"tmux_base" yaml:"tmux_base"`
	DefaultDepth            int    `mapstructure:"default_depth" yaml:"default_depth"`
	TmuxSessionPrefix       string `mapstructure:"tmux_session_prefix" yaml:"tmux_session_prefix"`
	AlwaysKillOnLastSession bool   `mapstructure:"always_kill_on_last_session" yaml:"always_kill_on_last_session"`
}

// Config represents the full configuration structure
type Config struct {
	ScanDirs        []ScanDir         `mapstructure:"scan_dirs" yaml:"scan_dirs"`
	EntryDirs       []string          `mapstructure:"entry_dirs" yaml:"entry_dirs"`
	IgnoreDirs      []string          `mapstructure:"ignore_dirs" yaml:"ignore_dirs"`
	PrimaryTemplate SessionTemplate   `mapstructure:"primary_template" yaml:"primary_template"`
	Templates       []SessionTemplate `mapstructure:"templates,omitempty" yaml:"templates,omitempty"`
	Settings        Settings          `mapstructure:"settings" yaml:"settings"`
}
