package config

import (
	"os"
	"strings"
)

// Editor returns the command that edits the config when "muxly config edit" isn't given
// one. settingsEditor is settings.editor, which MUXLY_EDITOR replaces when the config
// loads. After that come $VISUAL, $EDITOR and vi, the same order git uses. A blank value
// counts as not set.
func Editor(settingsEditor string) string {
	for _, candidate := range []string{settingsEditor, os.Getenv(envVisual), os.Getenv(envEditor)} {
		if strings.TrimSpace(candidate) != "" {
			return candidate
		}
	}
	return DefaultEditor
}
