// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package forms

import (
	"fmt"
	"strings"

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
			Title("New Session Title").
			Value(sessionName),
		huh.NewSelect[string]().
			Title("Base path").
			Options(huh.NewOptions("Home", "CWD", "Custom")...).
			Value(pathOption),
	).WithHideFunc(func() bool {
		return *useDefault
	})

	third := huh.NewGroup(
		huh.NewInput().
			Title("Custom Path").
			Value(customPath),
	).WithHideFunc(func() bool {
		return *pathOption != "Custom"
	})

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Session Name: %s\n", *sessionName))
	b.WriteString(fmt.Sprintf("Path Option: %s\n", *pathOption))
	if *pathOption == "Custom" {
		b.WriteString(fmt.Sprintf("Custom Path: %s\n", *customPath))
	}
	b.WriteString(fmt.Sprintf("Windows: %s", *windowStr))

	last := huh.NewGroup(
		huh.NewConfirm().
			Title("Create this session?").
			Description(b.String()).
			Value(confirmCreate),
	)

	return huh.NewForm(first, second, third, last).WithTheme(huh.ThemeBase())
}
