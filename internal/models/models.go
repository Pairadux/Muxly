// Package models defines muxly's config and session types.
package models

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// StringSet represents a set of strings using a map with empty struct values for memory efficiency
type StringSet map[string]struct{}

// Window is one tmux window in a layout, with an optional command to run in it.
type Window struct {
	Name string `mapstructure:"name" yaml:"name"`
	Cmd  string `mapstructure:"cmd,omitempty" yaml:"cmd,omitempty"`
}

// SessionLayout is the set of windows to create in a new session.
type SessionLayout struct {
	Windows []Window `mapstructure:"windows" yaml:"windows"`
}

// SessionTemplate is a named layout from the config, with an optional fixed path.
type SessionTemplate struct {
	Name    string   `mapstructure:"name" yaml:"name"`
	Label   string   `mapstructure:"label,omitempty" yaml:"label,omitempty"`
	Default bool     `mapstructure:"default,omitempty" yaml:"default,omitempty"`
	Path    string   `mapstructure:"path,omitempty" yaml:"path,omitempty"`
	Windows []Window `mapstructure:"windows" yaml:"windows"`
}

// ScanDir is a directory from scan_dirs, whose subdirectories become picker entries.
type ScanDir struct {
	Path     string `mapstructure:"path" yaml:"path"`
	Depth    *int   `mapstructure:"depth,omitempty" yaml:"depth,omitempty"`
	Alias    string `mapstructure:"alias,omitempty" yaml:"alias,omitempty"`
	Template string `mapstructure:"template,omitempty" yaml:"template,omitempty"`
}

// EntryDir is a directory from entry_dirs, which becomes a picker entry itself.
type EntryDir struct {
	Path     string `mapstructure:"path" yaml:"path"`
	Template string `mapstructure:"template,omitempty" yaml:"template,omitempty"`
}

// Session is everything needed to create one tmux session.
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
	if s.Template != "" {
		result = fmt.Sprintf("%s (template: %s)", result, s.Template)
	}
	return result
}

// String returns the string representation
func (e EntryDir) String() string {
	result := e.Path
	if e.Template != "" {
		result = fmt.Sprintf("%s (template: %s)", result, e.Template)
	}
	return result
}

// Settings groups general configuration options
type Settings struct {
	Editor                  string `mapstructure:"editor" yaml:"editor,omitempty"`
	DefaultDepth            int    `mapstructure:"default_depth" yaml:"default_depth"`
	TmuxSessionPrefix       string `mapstructure:"tmux_session_prefix" yaml:"tmux_session_prefix"`
	AlwaysKillOnLastSession bool   `mapstructure:"always_kill_on_last_session" yaml:"always_kill_on_last_session"`

	// RemovedTmuxBase catches a key muxly no longer uses, so that a config which still has
	// it gets a warning naming the line, rather than the error for an unknown key. Configs
	// generated before 0.27.0 all have it.
	RemovedTmuxBase yaml.Node `mapstructure:"-" yaml:"tmux_base,omitempty"`
}

// Config represents the full configuration structure
type Config struct {
	ScanDirs   []ScanDir         `mapstructure:"scan_dirs" yaml:"scan_dirs"`
	EntryDirs  []EntryDir        `mapstructure:"entry_dirs" yaml:"entry_dirs"`
	IgnoreDirs []string          `mapstructure:"ignore_dirs" yaml:"ignore_dirs"`
	Templates  []SessionTemplate `mapstructure:"templates" yaml:"templates"`
	Settings   Settings          `mapstructure:"settings" yaml:"settings"`
}
