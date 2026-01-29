package selector

import "testing"

func TestSanitizeSessionName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedName  string
		expectedCount int
	}{
		{"no change", "muxly", "muxly", 0},
		{"single leading dot stripped", ".config", "config", 1},
		{"two leading dots stripped", "..double", "double", 2},
		{"three leading dots stripped", "...triple", "triple", 3},
		{"just a dot", ".", "", 1},
		{"just two dots", "..", "", 2},
		{"empty string", "", "", 0},
		{"middle dot to underscore", "normal.name", "normal_name", 0},
		{"multiple middle dots to underscores", "no.leading.dots", "no_leading_dots", 0},
		{"colon to dash", "has:colon", "has-colon", 0},
		{"multiple colons to dashes", "multi:col:ons", "multi-col-ons", 0},
		{"leading dot stripped colon to dash", ".dot:colon", "dot-colon", 1},
		{"leading dot stripped middle dot to underscore", ".mid.dot", "mid_dot", 1},
		{"middle dot and colon", "all.chars:here", "all_chars-here", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotCount := SanitizeSessionName(tt.input)
			if gotName != tt.expectedName {
				t.Errorf("SanitizeSessionName(%q) name = %q, want %q", tt.input, gotName, tt.expectedName)
			}
			if gotCount != tt.expectedCount {
				t.Errorf("SanitizeSessionName(%q) count = %d, want %d", tt.input, gotCount, tt.expectedCount)
			}
		})
	}
}

func TestDotdirSuffix(t *testing.T) {
	tests := []struct {
		dotCount int
		expected string
	}{
		{0, ""},
		{1, " [dotdir]"},
		{2, " [dotdir x2]"},
		{3, " [dotdir x3]"},
		{10, " [dotdir x10]"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := DotdirSuffix(tt.dotCount)
			if got != tt.expected {
				t.Errorf("DotdirSuffix(%d) = %q, want %q", tt.dotCount, got, tt.expected)
			}
		})
	}
}

func TestSanitizePathForDisplay(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"simple name", "muxly", "muxly"},
		{"path with directories", "Dev/muxly", "Dev/muxly"},
		{"path with dotfile last component", "Dev/.config", "Dev/config"},
		{"dotfile not last component unchanged", ".config/nvim", ".config/nvim"},
		{"dotfile in middle and end", ".config/.local", ".config/local"},
		{"absolute path with double dot", "/absolute/..hidden", "/absolute/hidden"},
		{"middle dot in last component", "Dev/my.project", "Dev/my_project"},
		{"colon in last component", "Dev/has:colon", "Dev/has-colon"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizePathForDisplay(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizePathForDisplay(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestApplyPrefix(t *testing.T) {
	tests := []struct {
		prefix   string
		name     string
		expected string
	}{
		{"", "muxly", "muxly"},
		{"cfg", "muxly", "cfg/muxly"},
		{"my-alias", "project", "my-alias/project"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := ApplyPrefix(tt.prefix, tt.name)
			if got != tt.expected {
				t.Errorf("ApplyPrefix(%q, %q) = %q, want %q", tt.prefix, tt.name, got, tt.expected)
			}
		})
	}
}
