package forms

import (
	"github.com/Pairadux/muxly/internal/models"

	"github.com/charmbracelet/huh"
)

func TemplateSelectForm(templates []models.SessionTemplate, selectedIdx *int) *huh.Form {
	options := make([]huh.Option[int], len(templates))
	for i, tmpl := range templates {
		options[i] = huh.NewOption(tmpl.Name, i)
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
