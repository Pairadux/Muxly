package forms

import (
	"github.com/Pairadux/muxly/internal/models"

	"github.com/charmbracelet/huh"
)

func TemplateSelectForm(templates []models.SessionTemplate, selectedIdx *int) *huh.Form {
	options := make([]huh.Option[int], len(templates))
	for i, tmpl := range templates {
		displayName := tmpl.Name
		if tmpl.Label != "" {
			displayName = tmpl.Label
		}
		options[i] = huh.NewOption(displayName, i)
	}

	return huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select a template").
				Options(options...).
				Value(selectedIdx),
		),
	)
}

func ConfirmationForm(title, description string, confirm *bool) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Value(confirm),
		),
	)
}
