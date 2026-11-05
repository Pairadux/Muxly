package config

import "testing"

func TestEditor(t *testing.T) {
	tests := []struct {
		name     string
		settings string
		visual   string
		editor   string
		want     string
	}{
		{name: "nothing set", want: "vi"},
		{name: "EDITOR", editor: "e", want: "e"},
		{name: "VISUAL beats EDITOR", visual: "v", editor: "e", want: "v"},
		{name: "settings beat VISUAL and EDITOR", settings: "s", visual: "v", editor: "e", want: "s"},
		{name: "blank settings are skipped", settings: " ", visual: "v", want: "v"},
		{name: "blank VISUAL is skipped", visual: "\t", editor: "e", want: "e"},
		{name: "blank EDITOR is skipped", editor: " ", want: "vi"},
		{name: "arguments stay", editor: "code --wait", want: "code --wait"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)

			if got := Editor(tt.settings); got != tt.want {
				t.Errorf("Editor(%q) = %q, want %q", tt.settings, got, tt.want)
			}
		})
	}
}
