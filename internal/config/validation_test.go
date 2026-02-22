package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *models.Config
		expectError bool
		errContains string
	}{
		{
			name: "valid scan_dirs only",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: false,
		},
		{
			name: "valid entry_dirs only",
			cfg: &models.Config{
				EntryDirs: []models.EntryDir{{Path: "~/Documents"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: false,
		},
		{
			name: "valid both dirs",
			cfg: &models.Config{
				ScanDirs:  []models.ScanDir{{Path: "~/Dev"}},
				EntryDirs: []models.EntryDir{{Path: "~/Documents"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: false,
		},
		{
			name: "invalid no dirs",
			cfg: &models.Config{
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "no directories",
		},
		{
			name: "invalid no templates",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
			},
			expectError: true,
			errContains: "at least one template is required",
		},
		{
			name: "invalid default template no windows",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{}},
				},
			},
			expectError: true,
			errContains: "at least one window",
		},
		{
			name: "invalid template no name",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "all templates must have a name",
		},
		{
			name: "invalid no default template",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "exactly one template must have default: true",
		},
		{
			name: "invalid multiple default templates",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Name: "First", Default: true, Windows: []models.Window{{Name: "main"}}},
					{Name: "Second", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "only one template can have default: true",
		},
		{
			name: "invalid duplicate template name",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
					{Name: "Default", Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "duplicate template name",
		},
		{
			name: "invalid template no windows",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
					{Name: "Empty", Windows: []models.Window{}},
				},
			},
			expectError: true,
			errContains: "must have at least one window",
		},
		{
			name: "invalid extra template no name",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
					{Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "all templates must have a name",
		},
		{
			name: "valid with multiple templates",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
					{Name: "Dev", Windows: []models.Window{{Name: "editor"}, {Name: "term"}}},
				},
			},
			expectError: false,
		},
		{
			name: "invalid duplicate alias",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev", Alias: "myalias"},
					{Path: "~/Work", Alias: "myalias"},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "duplicate alias",
		},
		{
			name: "valid same path no alias",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev"},
					{Path: "~/Dev"},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: false,
		},
		{
			name: "valid empty alias strings",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev", Alias: ""},
					{Path: "~/Work", Alias: ""},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: false,
		},
		{
			name: "valid unique aliases",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev", Alias: "dev"},
					{Path: "~/Work", Alias: "work"},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: false,
		},
		{
			name: "valid scan_dir template reference",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev", Template: "Dev"},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
					{Name: "Dev", Windows: []models.Window{{Name: "editor"}}},
				},
			},
			expectError: false,
		},
		{
			name: "valid scan_dir references default template",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev", Template: "Default"},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: false,
		},
		{
			name: "invalid scan_dir unknown template",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev", Template: "Nonexistent"},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "scan_dir \"~/Dev\" references unknown template",
		},
		{
			name: "valid entry_dir template reference",
			cfg: &models.Config{
				EntryDirs: []models.EntryDir{
					{Path: "~/Documents", Template: "Dev"},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
					{Name: "Dev", Windows: []models.Window{{Name: "editor"}}},
				},
			},
			expectError: false,
		},
		{
			name: "invalid entry_dir unknown template",
			cfg: &models.Config{
				EntryDirs: []models.EntryDir{
					{Path: "~/Documents", Template: "Nonexistent"},
				},
				Templates: []models.SessionTemplate{
					{Name: "Default", Default: true, Windows: []models.Window{{Name: "main"}}},
				},
			},
			expectError: true,
			errContains: "entry_dir \"~/Documents\" references unknown template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.cfg)
			if tt.expectError {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateConfigFile(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setup       func() string
		expectError bool
		errContains string
	}{
		{
			name: "valid file",
			setup: func() string {
				path := filepath.Join(tempDir, "valid.yaml")
				content := `
scan_dirs:
  - path: ~/Dev
templates:
  - name: Default
    default: true
    windows:
      - name: main
`
				os.WriteFile(path, []byte(content), 0644)
				return path
			},
			expectError: false,
		},
		{
			name: "file not found",
			setup: func() string {
				return filepath.Join(tempDir, "nonexistent.yaml")
			},
			expectError: true,
			errContains: "cannot read config file",
		},
		{
			name: "invalid YAML",
			setup: func() string {
				path := filepath.Join(tempDir, "invalid_yaml.yaml")
				content := `
scan_dirs:
  - path: ~/Dev
  invalid yaml content here
    - this: is broken
`
				os.WriteFile(path, []byte(content), 0644)
				return path
			},
			expectError: true,
			errContains: "invalid YAML",
		},
		{
			name: "valid YAML invalid config",
			setup: func() string {
				path := filepath.Join(tempDir, "invalid_config.yaml")
				content := `
templates:
  - name: Default
    default: true
    windows:
      - name: main
`
				os.WriteFile(path, []byte(content), 0644)
				return path
			},
			expectError: true,
			errContains: "invalid config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup()
			cfg, err := ValidateConfigFile(path)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateConfigFile() expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ValidateConfigFile() error = %q, want error containing %q", err.Error(), tt.errContains)
				}
				if cfg != nil {
					t.Errorf("ValidateConfigFile() expected nil config on error, got %v", cfg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateConfigFile() unexpected error: %v", err)
				}
				if cfg == nil {
					t.Error("ValidateConfigFile() expected non-nil config, got nil")
				}
			}
		})
	}
}
