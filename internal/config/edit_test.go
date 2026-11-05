package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

const editTemplates = `templates:
  - name: default
    default: true
    windows:
      - name: main
`

func writeConfigFile(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readConfigFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestAddEntryDir(t *testing.T) {
	tests := []struct {
		name string
		path string
		text string
		want string
	}{
		{
			name: "after the last item, leaving blank lines, comments and indents alone",
			path: "~/new",
			text: `# My config
entry_dirs:
    - path: ~/a   # why a

    - path: ~/b
      template: default

    # - path: ~/old

# +-----------+
# | TEMPLATES |
# +-----------+
` + editTemplates + `settings:
    editor: nvim
    default_depth:   3
`,
			want: `# My config
entry_dirs:
    - path: ~/a   # why a

    - path: ~/b
      template: default
    - path: ~/new

    # - path: ~/old

# +-----------+
# | TEMPLATES |
# +-----------+
` + editTemplates + `settings:
    editor: nvim
    default_depth:   3
`,
		},
		{
			name: "a list with no indent",
			path: "~/new",
			text: "entry_dirs:\n- path: ~/a\n" + editTemplates,
			want: "entry_dirs:\n- path: ~/a\n- path: ~/new\n" + editTemplates,
		},
		{
			name: "the last line of a file with no final newline",
			path: "~/new",
			text: editTemplates + "entry_dirs:\n  - path: ~/a",
			want: editTemplates + "entry_dirs:\n  - path: ~/a\n  - path: ~/new\n",
		},
		{
			name: "an empty inline list, as config init writes",
			path: "~/new",
			text: "scan_dirs:\n  - path: ~/s\nentry_dirs: []\n" + editTemplates,
			want: "scan_dirs:\n  - path: ~/s\nentry_dirs:\n  - path: ~/new\n" + editTemplates,
		},
		{
			name: "a key with nothing after it",
			path: "~/new",
			text: "scan_dirs:\n  - path: ~/s\nentry_dirs:\n" + editTemplates,
			want: "scan_dirs:\n  - path: ~/s\nentry_dirs:\n  - path: ~/new\n" + editTemplates,
		},
		{
			name: "a null with a comment, which stays",
			path: "~/new",
			text: "scan_dirs:\n  - path: ~/s\nentry_dirs: ~ # none yet\n" + editTemplates,
			want: "scan_dirs:\n  - path: ~/s\nentry_dirs: # none yet\n  - path: ~/new\n" + editTemplates,
		},
		{
			name: "no key yet, at the end and indented like the rest of the file",
			path: "~/new",
			text: "scan_dirs:\n    - path: ~/s\n" + editTemplates,
			want: "scan_dirs:\n    - path: ~/s\n" + editTemplates + "entry_dirs:\n    - path: ~/new\n",
		},
		{
			name: "values that need quoting are quoted",
			path: "~",
			text: "entry_dirs:\n  - path: ~/a\n" + editTemplates,
			want: "entry_dirs:\n  - path: ~/a\n  - path: \"~\"\n" + editTemplates,
		},
		{
			name: "after a short-form item",
			path: "~/new",
			text: "entry_dirs:\n  - ~/a\n  - ~/b\n" + editTemplates,
			want: "entry_dirs:\n  - ~/a\n  - ~/b\n  - path: ~/new\n" + editTemplates,
		},
		{
			name: "after a path of an unquoted tilde, which means null to yaml",
			path: "~/new",
			text: "entry_dirs:\n  - path: ~\n" + editTemplates,
			want: "entry_dirs:\n  - path: ~\n  - path: ~/new\n" + editTemplates,
		},
		{
			name: "a tmux_base line stays",
			path: "~/new",
			text: "entry_dirs:\n  - path: ~/a\n" + editTemplates + "settings:\n  tmux_base: 1\n",
			want: "entry_dirs:\n  - path: ~/a\n  - path: ~/new\n" + editTemplates + "settings:\n  tmux_base: 1\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.text)

			if err := AddEntryDir(path, models.EntryDir{Path: tt.path}); err != nil {
				t.Fatal(err)
			}

			if got := readConfigFile(t, path); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestAddScanDirWritesEveryOption(t *testing.T) {
	depth := 2
	path := writeConfigFile(t, "entry_dirs:\n  - path: ~/a\nscan_dirs: []\n"+editTemplates)

	if err := AddScanDir(path, models.ScanDir{Path: "~/s", Depth: &depth, Alias: "s"}); err != nil {
		t.Fatal(err)
	}

	want := "entry_dirs:\n  - path: ~/a\nscan_dirs:\n  - path: ~/s\n    depth: 2\n    alias: s\n" + editTemplates
	if got := readConfigFile(t, path); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRemoveEntryDir(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		index int
		want  string
	}{
		{
			name:  "one line",
			text:  "entry_dirs:\n  - path: ~/a\n  - path: ~/b\n  - path: ~/c\n" + editTemplates,
			index: 1,
			want:  "entry_dirs:\n  - path: ~/a\n  - path: ~/c\n" + editTemplates,
		},
		{
			name:  "all of a longer item, with comments on their own lines staying",
			text:  "entry_dirs:\n    # about a\n    - path: ~/a\n      template: default # gone with it\n\n    # about b\n    - path: ~/b\n" + editTemplates,
			index: 0,
			want:  "entry_dirs:\n    # about a\n\n    # about b\n    - path: ~/b\n" + editTemplates,
		},
		{
			name:  "a short-form item",
			text:  "entry_dirs:\n  - ~/a\n  - ~/b\n" + editTemplates,
			index: 0,
			want:  "entry_dirs:\n  - ~/b\n" + editTemplates,
		},
		{
			name:  "the last item of a list when another list still has some",
			text:  "scan_dirs:\n  - path: ~/s\nentry_dirs:\n  - path: ~/a\n" + editTemplates,
			index: 0,
			want:  "scan_dirs:\n  - path: ~/s\nentry_dirs:\n" + editTemplates,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.text)

			if err := RemoveEntryDir(path, tt.index); err != nil {
				t.Fatal(err)
			}

			if got := readConfigFile(t, path); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestRemoveScanDir(t *testing.T) {
	path := writeConfigFile(t, "scan_dirs:\n  - path: ~/s\n    depth: 2\n  - path: ~/t\n"+editTemplates)

	if err := RemoveScanDir(path, 0); err != nil {
		t.Fatal(err)
	}

	want := "scan_dirs:\n  - path: ~/t\n" + editTemplates
	if got := readConfigFile(t, path); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

// Every one of these is written in a way the edit can't safely handle, so the only
// acceptable result is the file exactly as it was and a pointer to doing it by hand.
func TestEditsThatCannotBeMadeLeaveTheFileAlone(t *testing.T) {
	add := func(path string) error { return AddEntryDir(path, models.EntryDir{Path: "~/new"}) }
	remove := func(path string) error { return RemoveEntryDir(path, 0) }

	tests := []struct {
		name string
		text string
		edit func(string) error
		want string
	}{
		{name: "adding to an inline list", text: "entry_dirs: [{path: ~/a}]\n" + editTemplates, edit: add, want: "muxly config edit"},
		{name: "removing from an inline list", text: "scan_dirs:\n  - path: ~/s\nentry_dirs: [{path: ~/a}]\n" + editTemplates, edit: remove, want: "muxly config edit"},
		{name: "adding after a value over several lines", text: "entry_dirs:\n  - path: >-\n      ~/a\n" + editTemplates, edit: add, want: "muxly config edit"},
		{name: "removing a value over several lines", text: "scan_dirs:\n  - path: ~/s\nentry_dirs:\n  - path: >-\n      ~/a\n" + editTemplates, edit: remove, want: "muxly config edit"},
		{name: "adding after a dash on its own line", text: "entry_dirs:\n  -\n    path: ~/a\n" + editTemplates, edit: add, want: "muxly config edit"},
		{name: "removing a dash on its own line", text: "scan_dirs:\n  - path: ~/s\nentry_dirs:\n  -\n    path: ~/a\n" + editTemplates, edit: remove, want: "muxly config edit"},
		{name: "an empty inline list over two lines", text: "scan_dirs:\n  - path: ~/s\nentry_dirs: [\n  ]\n" + editTemplates, edit: add, want: "muxly config edit"},
		{name: "a value of the wrong kind", text: "scan_dirs:\n  - path: ~/s\nentry_dirs: 7\n" + editTemplates, edit: add, want: "invalid config file"},
		{name: "line breaks yaml counts but the edit doesn't", text: "entry_dirs:\r  - path: ~/a\rtemplates:\r  - name: default\r    default: true\r    windows:\r      - name: main\r", edit: add, want: "muxly config edit"},
		{name: "removing the last directory there is", text: "entry_dirs:\n  - path: ~/a\n" + editTemplates, edit: remove, want: "no directories configured"},
		{name: "a file muxly would refuse", text: "entry_dirs:\n  - path: ~/a\ntempaltes: []\n", edit: add, want: "unknown key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfigFile(t, tt.text)

			err := tt.edit(path)

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %v, want one mentioning %q", err, tt.want)
			}
			if got := readConfigFile(t, path); got != tt.text {
				t.Errorf("the file changed:\n%s", got)
			}
		})
	}
}

// A real shape: section banners, blank lines between items, a commented-out item, and
// four-space indents. Adding then removing has to give back exactly what was there.
func TestAddingThenRemovingRestoresTheFile(t *testing.T) {
	original := `# +-----------+
# | SCAN-DIRS |
# +-----------+
scan_dirs:
    - path: ~/.dotfiles/dot_config

    - path: ~/Projects
      template: "default"

    # - path: ~/Projects/CS-Homework

# +------------+
# | ENTRY-DIRS |
# +------------+
entry_dirs:
    - path: ~/.dotfiles
    - path: ~/Vault

# +-----------+
# | TEMPLATES |
# +-----------+
` + editTemplates + `
settings:
    editor: nvim
    tmux_base: 1
    tmux_session_prefix: '[TMUX] '
`
	path := writeConfigFile(t, original)

	if err := AddScanDir(path, models.ScanDir{Path: "~/new"}); err != nil {
		t.Fatal(err)
	}
	if err := AddEntryDir(path, models.EntryDir{Path: "~/new"}); err != nil {
		t.Fatal(err)
	}
	if got := readConfigFile(t, path); got == original {
		t.Fatal("nothing was added")
	}
	if err := RemoveScanDir(path, 2); err != nil {
		t.Fatal(err)
	}
	if err := RemoveEntryDir(path, 2); err != nil {
		t.Fatal(err)
	}

	if got := readConfigFile(t, path); got != original {
		t.Errorf("got:\n%s\nwant:\n%s", got, original)
	}
}
