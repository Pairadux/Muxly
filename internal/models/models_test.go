package models

import "testing"

func TestScanDirGetDepth(t *testing.T) {
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name         string
		scanDir      ScanDir
		flagDepth    int
		defaultDepth int
		expected     int
	}{
		{
			name:         "flag depth takes precedence",
			scanDir:      ScanDir{Path: "~/Dev", Depth: intPtr(3)},
			flagDepth:    5,
			defaultDepth: 2,
			expected:     5,
		},
		{
			name:         "config depth when no flag",
			scanDir:      ScanDir{Path: "~/Dev", Depth: intPtr(3)},
			flagDepth:    0,
			defaultDepth: 2,
			expected:     3,
		},
		{
			name:         "default depth when no flag or config",
			scanDir:      ScanDir{Path: "~/Dev"},
			flagDepth:    0,
			defaultDepth: 2,
			expected:     2,
		},
		{
			name:         "fallback to 1 when all zero",
			scanDir:      ScanDir{Path: "~/Dev"},
			flagDepth:    0,
			defaultDepth: 0,
			expected:     1,
		},
		{
			name:         "flag depth 0 is not used",
			scanDir:      ScanDir{Path: "~/Dev", Depth: intPtr(3)},
			flagDepth:    0,
			defaultDepth: 2,
			expected:     3,
		},
		{
			name:         "config depth 0 is used",
			scanDir:      ScanDir{Path: "~/Dev", Depth: intPtr(0)},
			flagDepth:    0,
			defaultDepth: 2,
			expected:     0,
		},
		{
			name:         "negative flag depth is not used",
			scanDir:      ScanDir{Path: "~/Dev", Depth: intPtr(3)},
			flagDepth:    -1,
			defaultDepth: 2,
			expected:     3,
		},
		{
			name:         "negative default depth is not used",
			scanDir:      ScanDir{Path: "~/Dev"},
			flagDepth:    0,
			defaultDepth: -1,
			expected:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scanDir.GetDepth(tt.flagDepth, tt.defaultDepth)
			if got != tt.expected {
				t.Errorf("GetDepth(%d, %d) = %d, want %d", tt.flagDepth, tt.defaultDepth, got, tt.expected)
			}
		})
	}
}

func TestScanDirString(t *testing.T) {
	intPtr := func(i int) *int { return &i }

	tests := []struct {
		name     string
		scanDir  ScanDir
		expected string
	}{
		{
			name:     "path only",
			scanDir:  ScanDir{Path: "~/Dev"},
			expected: "~/Dev",
		},
		{
			name:     "path with depth",
			scanDir:  ScanDir{Path: "~/Dev", Depth: intPtr(2)},
			expected: "~/Dev:2",
		},
		{
			name:     "path with alias",
			scanDir:  ScanDir{Path: "~/Dev", Alias: "dev"},
			expected: "~/Dev (alias: dev)",
		},
		{
			name:     "path with depth and alias",
			scanDir:  ScanDir{Path: "~/Dev", Depth: intPtr(3), Alias: "dev"},
			expected: "~/Dev:3 (alias: dev)",
		},
		{
			name:     "depth of 0",
			scanDir:  ScanDir{Path: "~/projects", Depth: intPtr(0)},
			expected: "~/projects:0",
		},
		{
			name:     "empty alias is not shown",
			scanDir:  ScanDir{Path: "~/Dev", Alias: ""},
			expected: "~/Dev",
		},
		{
			name:     "path with template",
			scanDir:  ScanDir{Path: "~/Dev", Template: "Go Dev"},
			expected: "~/Dev (template: Go Dev)",
		},
		{
			name:     "path with depth alias and template",
			scanDir:  ScanDir{Path: "~/Dev", Depth: intPtr(2), Alias: "dev", Template: "Go Dev"},
			expected: "~/Dev:2 (alias: dev) (template: Go Dev)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scanDir.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestEntryDirString(t *testing.T) {
	tests := []struct {
		name     string
		entryDir EntryDir
		expected string
	}{
		{
			name:     "path only",
			entryDir: EntryDir{Path: "~/Documents"},
			expected: "~/Documents",
		},
		{
			name:     "path with template",
			entryDir: EntryDir{Path: "~/Documents", Template: "Single Window"},
			expected: "~/Documents (template: Single Window)",
		},
		{
			name:     "empty template is not shown",
			entryDir: EntryDir{Path: "~/Documents", Template: ""},
			expected: "~/Documents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entryDir.String()
			if got != tt.expected {
				t.Errorf("String() = %q, want %q", got, tt.expected)
			}
		})
	}
}
