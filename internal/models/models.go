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
	Name string `yaml:"name"`
	Cmd  string `yaml:"cmd,omitempty"`
}

// SessionLayout is the set of windows to create in a new session.
type SessionLayout struct {
	Windows []Window `yaml:"windows"`
}

// SessionTemplate is a named layout from the config, with an optional fixed path.
type SessionTemplate struct {
	Name    string   `yaml:"name"`
	Label   string   `yaml:"label,omitempty"`
	Default bool     `yaml:"default,omitempty"`
	Path    string   `yaml:"path,omitempty"`
	Windows []Window `yaml:"windows"`
}

// ScanDir is a directory from scan_dirs, whose subdirectories become picker entries.
type ScanDir struct {
	Path     string `yaml:"path"`
	Depth    *int   `yaml:"depth,omitempty"`
	Alias    string `yaml:"alias,omitempty"`
	Template string `yaml:"template,omitempty"`
}

// EntryDir is a directory from entry_dirs, which becomes a picker entry itself.
type EntryDir struct {
	Path     string `yaml:"path"`
	Template string `yaml:"template,omitempty"`
}

// Session is everything needed to create one tmux session.
type Session struct {
	Name   string
	Path   string
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

// UnmarshalYAML accepts "- ~/notes" as well as "- path: ~/notes". Decoding from a node isn't
// strict about keys, so config.Parse checks those.
func (e *EntryDir) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		return value.Decode(&e.Path)
	case yaml.MappingNode:
		type plain EntryDir
		return value.Decode((*plain)(e))
	}
	return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: an entry_dirs item must be a path, or a mapping with a path key", value.Line)}}
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
	Editor                  string `yaml:"editor,omitempty"`
	DefaultDepth            int    `yaml:"default_depth"`
	TmuxSessionPrefix       string `yaml:"tmux_session_prefix"`
	AlwaysKillOnLastSession bool   `yaml:"always_kill_on_last_session"`

	// RemovedTmuxBase catches a key muxly no longer uses, so that a config which still has
	// it gets a warning naming the line, rather than the error for an unknown key. Configs
	// generated before 0.27.0 all have it.
	RemovedTmuxBase yaml.Node `yaml:"tmux_base,omitempty"`
}

// Config represents the full configuration structure
type Config struct {
	ScanDirs   []ScanDir         `yaml:"scan_dirs"`
	EntryDirs  []EntryDir        `yaml:"entry_dirs"`
	IgnoreDirs []string          `yaml:"ignore_dirs"`
	Templates  []SessionTemplate `yaml:"templates"`
	Settings   Settings          `yaml:"settings"`
}
