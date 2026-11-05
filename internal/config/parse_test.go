package config

import (
	"errors"
	"strings"
	"testing"
)

func TestParseRejectsWhatItDoesNotKnow(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		want        []string
		wantPartial bool
	}{
		{
			name:        "misspelled keys, at any level, with their lines",
			text:        "tempaltes:\n  - name: a\nsettings:\n  nope: 1\nscan_dirs:\n  - path: ~/a\n    alais: b\n",
			want:        []string{`line 1: unknown key "tempaltes"`, `line 4: unknown key "nope"`, `line 7: unknown key "alais"`},
			wantPartial: true,
		},
		{
			name:        "a key in the wrong case",
			text:        "SETTINGS:\n  editor: vi\n",
			want:        []string{`line 1: unknown key "SETTINGS"`},
			wantPartial: true,
		},
		{
			name:        "a value of the wrong type",
			text:        "settings:\n  default_depth: \"3\"\n",
			want:        []string{"line 2:", "into int"},
			wantPartial: true,
		},
		{
			name:        "a key given twice",
			text:        "settings:\n  editor: a\nsettings:\n  editor: b\n",
			want:        []string{"line 3:", `"settings" already defined at line 1`},
			wantPartial: true,
		},
		{
			name:        "a second document",
			text:        "settings:\n  editor: a\n---\nentry_dirs:\n  - path: ~/x\n",
			want:        []string{"line 3: a second YAML document"},
			wantPartial: true,
		},
		{
			name: "a syntax error",
			text: "settings:\n  editor: a\n bad: [\n",
			want: []string{"invalid YAML: line"},
		},
		{
			name: "a syntax error in a second document",
			text: "settings:\n  editor: a\n---\nbad: [\n",
			want: []string{"invalid YAML: line"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Parse([]byte(tt.text))

			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %v, want a ParseError", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q doesn't mention %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "models.") {
				t.Errorf("error %q shows an internal type", err)
			}
			if parseErr.Partial != tt.wantPartial {
				t.Errorf("Partial = %v, want %v", parseErr.Partial, tt.wantPartial)
			}
		})
	}
}

func TestParseKeepsWhatItCouldRead(t *testing.T) {
	cfg, _, err := Parse([]byte("tempaltes: []\nsettings:\n  editor: nvim\nentry_dirs:\n  - path: ~/a\n"))

	if err == nil {
		t.Fatal("want an error about the misspelled key")
	}
	if cfg.Settings.Editor != "nvim" || len(cfg.EntryDirs) != 1 {
		t.Errorf("config = %+v", cfg)
	}
}

func TestParseWarnsAboutTmuxBaseAndCarriesOn(t *testing.T) {
	cfg, warnings, err := Parse([]byte("entry_dirs:\n  - path: ~/a\nsettings:\n  editor: nvim\n  tmux_base: 1\n"))

	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "tmux_base on line 5 was removed") {
		t.Errorf("warnings = %q", warnings)
	}
	if cfg.Settings.Editor != "nvim" || len(cfg.EntryDirs) != 1 {
		t.Errorf("config = %+v", cfg)
	}
}

func TestParseWarnsAboutTmuxBaseEvenWithAnError(t *testing.T) {
	_, warnings, err := Parse([]byte("nope: 1\nsettings:\n  tmux_base: 1\n"))

	if err == nil || len(warnings) != 1 {
		t.Errorf("error = %v, warnings = %q", err, warnings)
	}
}

func TestParseOfNothingIsAnEmptyConfig(t *testing.T) {
	for _, text := range []string{"", "# only a comment\n", "---\n"} {
		cfg, warnings, err := Parse([]byte(text))

		if err != nil || warnings != nil {
			t.Errorf("Parse(%q): error = %v, warnings = %q", text, err, warnings)
		}
		if len(cfg.EntryDirs) != 0 || len(cfg.Templates) != 0 {
			t.Errorf("Parse(%q) = %+v", text, cfg)
		}
	}
}
