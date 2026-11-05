package selector

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

// makeDirs creates directories under a temp root and returns the root.
func makeDirs(t *testing.T, rel ...string) string {
	t.Helper()
	root := t.TempDir()
	for _, r := range rel {
		if err := os.MkdirAll(filepath.Join(root, r), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func labelsOf(dirs []Directory) []string {
	labels := make([]string, len(dirs))
	for i, dir := range dirs {
		labels[i] = dir.Label
	}
	slices.Sort(labels)
	return labels
}

func TestDirectoriesConfiguredTwiceAppearOnce(t *testing.T) {
	root := makeDirs(t, "dev/proj")
	proj := filepath.Join(root, "dev/proj")

	tests := []struct {
		name string
		cfg  models.Config
	}{
		{
			name: "scan_dirs and entry_dirs",
			cfg: models.Config{
				ScanDirs:  []models.ScanDir{{Path: filepath.Join(root, "dev")}},
				EntryDirs: []models.EntryDir{{Path: proj}},
			},
		},
		{
			name: "entry_dirs twice, one with a trailing slash",
			cfg:  models.Config{EntryDirs: []models.EntryDir{{Path: proj}, {Path: proj + "/"}}},
		},
		{
			name: "two overlapping scan_dirs",
			cfg: models.Config{ScanDirs: []models.ScanDir{
				{Path: filepath.Join(root, "dev")},
				{Path: root, Depth: new(2)},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirs := Directories(&tt.cfg, 0, false)

			matches := 0
			for _, dir := range dirs {
				if dir.Path == proj {
					matches++
					if dir.Label != "proj" {
						t.Errorf("label = %q, want proj", dir.Label)
					}
				}
			}
			if matches != 1 {
				t.Errorf("%s appears %d times in %+v", proj, matches, dirs)
			}
		})
	}
}

func TestDirectoriesEntryTemplateBeatsScanTemplate(t *testing.T) {
	root := makeDirs(t, "dev/proj", "dev/other")
	cfg := models.Config{
		ScanDirs: []models.ScanDir{{Path: filepath.Join(root, "dev"), Template: "from-scan", Alias: "d"}},
		EntryDirs: []models.EntryDir{
			{Path: filepath.Join(root, "dev/proj"), Template: "from-entry"},
			{Path: filepath.Join(root, "dev/other")},
		},
	}

	dirs := Directories(&cfg, 0, false)

	want := map[string]string{"d/proj": "from-entry", "d/other": "from-scan"}
	for _, dir := range dirs {
		if template, listed := want[dir.Label]; listed && dir.Template != template {
			t.Errorf("%s has template %q, want %q", dir.Label, dir.Template, template)
		}
	}
	if len(dirs) != 2 {
		t.Errorf("dirs = %+v", dirs)
	}
}

// Every directory in the fixtures that exist for trying this by hand, so a name that
// clashes in a new way can't silently cost someone an entry.
func TestDirectoriesLoseNothingFromTheFixtures(t *testing.T) {
	fixtures, err := filepath.Abs("../../testdata/muxly-fixtures")
	if err != nil {
		t.Fatal(err)
	}
	scanDir := func(rel string) models.ScanDir { return models.ScanDir{Path: filepath.Join(fixtures, rel)} }
	cfg := models.Config{ScanDirs: []models.ScanDir{
		scanDir("no-conflicts"),
		scanDir("same-basename/dev"),
		scanDir("same-basename/work"),
		scanDir("dotfiles"),
		scanDir("dotfile-conflicts/hidden"),
		scanDir("dotfile-conflicts/visible"),
		scanDir("special-chars"),
		{Path: filepath.Join(fixtures, "aliases/cfg-source"), Alias: "cfg"},
		{Path: filepath.Join(fixtures, "aliases/dev-source"), Alias: "dev"},
		scanDir("deep-nesting/a/b/c"),
	}}

	got := labelsOf(Directories(&cfg, 0, false))

	want := []string{
		"...triple", "..double-hidden", ".config", ".settings",
		"app:v2", "cfg/muxly", "dev/muxly", "dev/src", "my.project",
		"project", "project-alpha", "project-beta", "settings", "weird.name:here", "work/src",
	}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("labels = %q\nwant %q", got, want)
	}
}

func TestDirectoriesRespectDepth(t *testing.T) {
	root := makeDirs(t, "one/two/three")
	cfg := models.Config{ScanDirs: []models.ScanDir{{Path: root, Depth: new(2)}}}

	if got, want := labelsOf(Directories(&cfg, 0, false)), []string{"one", "two"}; !slices.Equal(got, want) {
		t.Errorf("configured depth: labels = %q, want %q", got, want)
	}
	if got, want := labelsOf(Directories(&cfg, 3, false)), []string{"one", "three", "two"}; !slices.Equal(got, want) {
		t.Errorf("flag depth: labels = %q, want %q", got, want)
	}
}

func TestDirectoriesIgnore(t *testing.T) {
	root := makeDirs(t, "dev/keep", "dev/target/inside", "dev/old/inside", "dev/.git/hooks")
	cfg := models.Config{
		ScanDirs:   []models.ScanDir{{Path: filepath.Join(root, "dev"), Depth: new(3)}},
		IgnoreDirs: []string{"target", filepath.Join(root, "dev/old")},
	}

	if got, want := labelsOf(Directories(&cfg, 0, false)), []string{"keep"}; !slices.Equal(got, want) {
		t.Errorf("labels = %q, want %q", got, want)
	}
}

func TestDirectoriesIgnoringAScanRootHidesAllOfIt(t *testing.T) {
	root := makeDirs(t, "dev/proj")
	cfg := models.Config{
		ScanDirs:   []models.ScanDir{{Path: filepath.Join(root, "dev")}},
		IgnoreDirs: []string{filepath.Join(root, "dev")},
	}

	if got := Directories(&cfg, 0, false); len(got) != 0 {
		t.Errorf("dirs = %+v, want none", got)
	}
}
