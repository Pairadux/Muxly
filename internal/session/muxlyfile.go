package session

import (
	"os"
	"path/filepath"

	"github.com/Pairadux/muxly/internal/models"
	"gopkg.in/yaml.v3"
)

// LoadMuxlyFile attempts to load a .muxly file from the given directory.
//
// Returns the parsed SessionLayout if the file exists and is valid YAML,
// or an empty SessionLayout otherwise. This provides project-specific
// session configuration that overrides the global session_layout from config.
//
// Errors are silently ignored since .muxly files are optional overrides.
func LoadMuxlyFile(path string) models.SessionLayout {
	layoutPath := filepath.Join(path, ".muxly")

	data, err := os.ReadFile(layoutPath)
	if err != nil {
		return models.SessionLayout{}
	}

	var layout models.SessionLayout
	if err := yaml.Unmarshal(data, &layout); err != nil {
		return models.SessionLayout{}
	}

	return layout
}
