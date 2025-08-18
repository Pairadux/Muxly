// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package forms

import (
	"fmt"
	"os"
	"strings"

	"github.com/Pairadux/Tmux-Sessionizer/internal/utility"

	"github.com/charmbracelet/huh"
	"github.com/mitchellh/go-homedir"
)

// CreateForm creates and returns an interactive form for session creation.
//
// The form collects user input for creating a new tmux session, including
// whether to use default settings, session name, path options, and window configuration.
// All parameters are pointers that will be populated with user selections.
func CreateForm(useFallback, confirmCreate *bool, sessionName, path, windowStr *string) *huh.Form {
	var (
		pathOption string
		customPath string
	)

	useFallbackGroup := huh.NewGroup(
		huh.NewConfirm().
			Title("Use Default Session?").
			Value(useFallback),
	)

	basicInfoGroup := huh.NewGroup(
		huh.NewInput().
			Inline(true).
			Title("New Session Title").
			Value(sessionName),
		huh.NewSelect[string]().
			Title("Base path").
			Options(huh.NewOptions("Home", "CWD", "Custom")...).
			Value(&pathOption),
	).WithHideFunc(func() bool {
		return *useFallback
	})

	customPathGroup := huh.NewGroup(
		huh.NewInput().
			Title("Custom Path").
			Description("Use ~ for home directory, or absolute/relative paths").
			Placeholder("~/Documents/projects").
			Value(&customPath).
			Validate(func(s string) error {
				_, err := utility.ResolvePath(s)
				return err
			}),
	).WithHideFunc(func() bool {
		return pathOption != "Custom"
	})

	sessionLayoutGroup := huh.NewGroup(
		// one window per line
		// FIXME: need to make the description show the defaults if nothing is input here
		// Might do a check to make sure that it has values so that the user HAS to input something
		huh.NewText().
			Title("Session Layout").
			Description("One window per line in the following format: name:cmd\nleave cmd empty for no cmd").
			Value(windowStr).
			ShowLineNumbers(true),
	).WithHideFunc(func() bool {
		return *useFallback
	})

	confirmGroup := huh.NewGroup(
		huh.NewConfirm().
			Title("Create this session?").
			DescriptionFunc(func() string {
				if *useFallback {
					return "Default Session"
				}

				var b strings.Builder

				b.WriteString(fmt.Sprintf("Session Name: %s\n", *sessionName))
				
				// TODO: need to show default values when fallback selected or nothing input in layout fields

				var err error
				*path, err = resolvePathOption(pathOption, customPath)
				if err != nil {
					*path = "[Error: " + err.Error() + "]"
				}
				b.WriteString(fmt.Sprintf("Path: %s\n", *path))

				if *windowStr != "" {
					b.WriteString("Windows:\n")
					for _, line := range strings.Split(*windowStr, "\n") {
						if strings.TrimSpace(line) != "" {
							b.WriteString(fmt.Sprintf("\t%s\n", strings.TrimSpace(line)))
						}
					}
				} else {
					b.WriteString("Windows: [Using default layout]\n")
				}
				return b.String()
			}, []any{sessionName, pathOption, customPath}).
			Value(confirmCreate),
	)

	return huh.NewForm(
		useFallbackGroup,
		basicInfoGroup,
		customPathGroup,
		sessionLayoutGroup,
		confirmGroup,
	) /*.WithTheme(huh.ThemeBase())*/
}

func resolvePathOption(pathOption, customPath string) (string, error) {
	switch pathOption {
	case "Home":
		return homedir.Dir()
	case "CWD":
		return os.Getwd()
	case "Custom":
		return utility.ResolvePath(customPath)
	}
	return "", fmt.Errorf("not sure what happened")
}
