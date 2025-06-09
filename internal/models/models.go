// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package models

type Window struct {
	Name string `yaml:"name" mapstructure:"name"`
	Cmd  string `yaml:"cmd,omitempty" mapstructure:"cmd,omitempty"`
}

type SessionLayout struct {
	Windows []Window `yaml:"windows" mapstructure:"windows"`
}

type ScanDir struct {
	Path  string `yaml:"path" mapstructure:"path"`
	Depth *int   `yaml:"depth,omitempty" mapstructure:"depth,omitempty"`
}

// type Entry struct {
// 	Path  string `yaml:"path" mapstructure:"path"`
// 	Depth int    `yaml:"depth" mapstructure:"depth"`
// }
