package config

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
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

func TestEntryDirsCanBeJustAPath(t *testing.T) {
	cfg, _, err := Parse([]byte("entry_dirs:\n  - ~/a\n  - path: ~/b\n    template: t\n  - \"~/c d\"\n"))

	if err != nil {
		t.Fatal(err)
	}
	want := []models.EntryDir{{Path: "~/a"}, {Path: "~/b", Template: "t"}, {Path: "~/c d"}}
	if !slices.Equal(cfg.EntryDirs, want) {
		t.Errorf("entry_dirs = %+v, want %+v", cfg.EntryDirs, want)
	}
}

func TestEntryDirsStayStrict(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{name: "an unknown key", text: "entry_dirs:\n  - path: ~/a\n    tempalte: t\n", want: []string{`line 3: unknown key "tempalte"`}},
		{name: "a misspelled path", text: "entry_dirs:\n  - pth: ~/a\n", want: []string{`line 2: unknown key "pth"`}},
		{name: "several, with others elsewhere", text: "entry_dirs:\n  - pth: ~/a\n  - path: ~/b\n    nope: 1\nsettings:\n  edtior: x\n", want: []string{`line 2: unknown key "pth"`, `line 4: unknown key "nope"`, `line 6: unknown key "edtior"`}},
		{name: "a list inside the list", text: "entry_dirs:\n  - - ~/a\n", want: []string{"line 2: an entry_dirs item must be a path"}},
		{name: "a value of the wrong type", text: "entry_dirs:\n  - path: [a]\n", want: []string{"line 2:"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Parse([]byte(tt.text))

			if err == nil {
				t.Fatal("want an error")
			}
			for _, want := range tt.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("%q doesn't mention %q", err, want)
				}
			}
		})
	}
}

// An unquoted ~ is YAML for null, which isn't what anyone writing it means.
func TestAnUnquotedTildeForAPathIsTheHomeDirectory(t *testing.T) {
	cfg, _, err := Parse([]byte("entry_dirs:\n  - path: ~\nscan_dirs:\n  - path: ~\n    depth: 2\n"))

	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.EntryDirs) != 1 || cfg.EntryDirs[0].Path != "~" {
		t.Errorf("entry_dirs = %+v", cfg.EntryDirs)
	}
	if len(cfg.ScanDirs) != 1 || cfg.ScanDirs[0].Path != "~" {
		t.Errorf("scan_dirs = %+v", cfg.ScanDirs)
	}
}

// The decoder would drop these with no error, so a directory would just go missing.
func TestEmptyListItemsAreErrors(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{name: "a bare tilde in entry_dirs", text: "entry_dirs:\n  - ~/a\n  - ~\n", want: "line 3: an empty list item"},
		{name: "a bare dash in entry_dirs", text: "entry_dirs:\n  - \n  - ~/a\n", want: "line 2: an empty list item"},
		{name: "scan_dirs", text: "scan_dirs:\n  - path: ~/a\n  - ~\n", want: "line 3: an empty list item"},
		{name: "ignore_dirs", text: "ignore_dirs:\n  - ~\n", want: "line 2: an empty list item"},
		{name: "windows", text: "templates:\n  - name: a\n    windows:\n      - \n", want: "line 4: an empty list item"},
		{name: "null", text: "entry_dirs:\n  - null\n", want: "line 2: an empty list item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Parse([]byte(tt.text))

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want one mentioning %q", err, tt.want)
			}
		})
	}
}

func TestAQuotedTildeIsAnEntry(t *testing.T) {
	cfg, _, err := Parse([]byte("entry_dirs:\n  - \"~\"\n"))

	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.EntryDirs) != 1 || cfg.EntryDirs[0].Path != "~" {
		t.Errorf("entry_dirs = %+v", cfg.EntryDirs)
	}
}

func TestParseReportsEveryKindOfProblemTogether(t *testing.T) {
	_, _, err := Parse([]byte("nope: 1\nentry_dirs:\n  - ~\n---\nsecond: doc\n"))

	if err == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{`line 1: unknown key "nope"`, "line 3: an empty list item", "line 4: a second YAML document"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("%q doesn't mention %q", err, want)
		}
	}
}
