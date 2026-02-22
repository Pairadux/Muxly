package config

import "github.com/Pairadux/muxly/internal/models"

// DefaultTemplate returns the template marked as default in the config.
// Returns false if no template has Default set.
func DefaultTemplate(cfg *models.Config) (models.SessionTemplate, bool) {
	for _, tmpl := range cfg.Templates {
		if tmpl.Default {
			return tmpl, true
		}
	}
	return models.SessionTemplate{}, false
}

// FindTemplateByName returns the first template matching the given name.
func FindTemplateByName(cfg *models.Config, name string) (models.SessionTemplate, bool) {
	for _, tmpl := range cfg.Templates {
		if tmpl.Name == name {
			return tmpl, true
		}
	}
	return models.SessionTemplate{}, false
}
