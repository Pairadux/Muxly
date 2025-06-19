// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package forms

import (
	// "errors"

	"github.com/charmbracelet/huh"
)

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

	return huh.NewForm(first, second, third).WithTheme(huh.ThemeBase())
}
