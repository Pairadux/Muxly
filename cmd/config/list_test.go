package config

import (
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

func TestFormatDirectories(t *testing.T) {
	depth := 3

	tests := []struct {
		name string
		cfg  models.Config
		want string
	}{
		{
			name: "empty sections say so",
			want: `scan_dirs:
  (none)
entry_dirs:
  (none)
ignore_dirs:
  (none)
`,
		},
		{
			name: "paths are shown as written, with every option",
			cfg: models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev", Depth: &depth, Alias: "dev", Template: "minimal"},
					{Path: "/srv/code"},
				},
				EntryDirs: []models.EntryDir{
					{Path: "~/notes"},
					{Path: "~/special", Template: "minimal"},
				},
				IgnoreDirs: []string{"target", "~/Dev/old"},
				Settings:   models.Settings{DefaultDepth: 2},
			},
			want: `scan_dirs:
  ~/Dev (depth 3, alias dev, template minimal)
  /srv/code (depth 2)
entry_dirs:
  ~/notes
  ~/special (template minimal)
ignore_dirs:
  target
  ~/Dev/old
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatDirectories(&tt.cfg); got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}
