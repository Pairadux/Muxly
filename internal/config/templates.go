package config

import "github.com/Pairadux/muxly/internal/models"

func FindTemplateByName(cfg *models.Config, name string) (models.SessionTemplate, bool) {
	if cfg.PrimaryTemplate.Name == name {
		return cfg.PrimaryTemplate, true
	}
	for _, tmpl := range cfg.Templates {
		if tmpl.Name == name {
			return tmpl, true
		}
	}
	return models.SessionTemplate{}, false
}
