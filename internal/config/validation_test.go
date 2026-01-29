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
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{{Name: "main"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid entry_dirs only",
			cfg: &models.Config{
				EntryDirs: []string{"~/Documents"},
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{{Name: "main"}},
				},
			},
			expectError: false,
		},
		{
			name: "valid both dirs",
			cfg: &models.Config{
				ScanDirs:  []models.ScanDir{{Path: "~/Dev"}},
				EntryDirs: []string{"~/Documents"},
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{{Name: "main"}},
				},
			},
			expectError: false,
		},
		{
			name: "invalid no dirs",
			cfg: &models.Config{
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{{Name: "main"}},
				},
			},
			expectError: true,
			errContains: "no directories",
		},
		{
			name: "invalid no windows",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{{Path: "~/Dev"}},
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{},
				},
			},
			expectError: true,
			errContains: "at least one window",
		},
		{
			name: "invalid duplicate alias",
			cfg: &models.Config{
				ScanDirs: []models.ScanDir{
					{Path: "~/Dev", Alias: "myalias"},
					{Path: "~/Work", Alias: "myalias"},
				},
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{{Name: "main"}},
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
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{{Name: "main"}},
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
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{{Name: "main"}},
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
				SessionLayout: models.SessionLayout{
					Windows: []models.Window{{Name: "main"}},
				},
			},
			expectError: false,
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
session_layout:
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
session_layout:
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
