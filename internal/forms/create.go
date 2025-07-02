// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package forms

import (
	// "errors"

	"github.com/charmbracelet/huh"
)

// CreateForm creates and returns an interactive form for session creation.
//
// The form collects user input for creating a new tmux session, including
// whether to use default settings, session name, path options, and window configuration.
// All parameters are pointers that will be populated with user selections.
func CreateForm(useDefault, confirmCreate *bool, sessionName, pathOption, customPath, windowStr *string) *huh.Form {
	first := huh.NewGroup(
		huh.NewConfirm().
			Title("Use Default Session?").
			Value(useDefault),
	)

	second := huh.NewGroup(
		huh.NewInput().
			Inline(true).
			Title("New Session Title"),
		huh.NewSelect[string]().
			Title("Base path").Options(
			huh.NewOptions("Home", "CWD", "Custom")...,
		).Value(pathOption),
	).WithHideFunc(func() bool {
		return *useDefault
	})

	// if useDefault {
	// 	// do something
	// }

	third := huh.NewGroup(
		huh.NewConfirm().
			Title("Example"),
	).WithHideFunc(func() bool {
		return *pathOption != "Custom"
	})

	last := huh.NewGroup(
		huh.NewConfirm().
			Title("Create this session?"),
	)

	return huh.NewForm(first, second, third, last).WithTheme(huh.ThemeBase())
}
