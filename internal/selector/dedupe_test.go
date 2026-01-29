package selector

import (
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

func TestDeduplicateDisplayNames(t *testing.T) {
	tests := []struct {
		name     string
		input    []models.PathInfo
		expected map[string]string
	}{
		{
			name:     "empty input",
			input:    []models.PathInfo{},
			expected: map[string]string{},
		},
		{
			name: "single directory",
			input: []models.PathInfo{
				{Path: "/home/user/Dev/muxly"},
			},
			expected: map[string]string{
				"/home/user/Dev/muxly": "muxly",
			},
		},
		{
			name: "two unique basenames",
			input: []models.PathInfo{
				{Path: "/Dev/foo"},
				{Path: "/Work/bar"},
			},
			expected: map[string]string{
				"/Dev/foo":  "foo",
				"/Work/bar": "bar",
			},
		},
		{
			name: "same basename different parents",
			input: []models.PathInfo{
				{Path: "/Dev/muxly"},
				{Path: "/Work/muxly"},
			},
			expected: map[string]string{
				"/Dev/muxly":  "Dev/muxly",
				"/Work/muxly": "Work/muxly",
			},
		},
		{
			name: "same basename deeper conflict",
			input: []models.PathInfo{
				{Path: "/a/b/src"},
				{Path: "/x/y/src"},
			},
			expected: map[string]string{
				"/a/b/src": "b/src",
				"/x/y/src": "y/src",
			},
		},
		{
			name: "same parent different grandparent",
			input: []models.PathInfo{
				{Path: "/a/shared/src"},
				{Path: "/b/shared/src"},
			},
			expected: map[string]string{
				"/a/shared/src": "a/shared/src",
				"/b/shared/src": "b/shared/src",
			},
		},
		{
			name: "three-way conflict",
			input: []models.PathInfo{
				{Path: "/Dev/muxly"},
				{Path: "/Work/muxly"},
				{Path: "/Projects/muxly"},
			},
			expected: map[string]string{
				"/Dev/muxly":      "Dev/muxly",
				"/Work/muxly":     "Work/muxly",
				"/Projects/muxly": "Projects/muxly",
			},
		},
		{
			name: "single with alias no conflict",
			input: []models.PathInfo{
				{Path: "/config/muxly", Prefix: "cfg"},
			},
			expected: map[string]string{
				"/config/muxly": "muxly",
			},
		},
		{
			name: "conflict one has alias",
			input: []models.PathInfo{
				{Path: "/config/muxly", Prefix: "cfg"},
				{Path: "/Dev/muxly"},
			},
			expected: map[string]string{
				"/config/muxly": "cfg/muxly",
				"/Dev/muxly":    "muxly",
			},
		},
		{
			name: "conflict both have aliases",
			input: []models.PathInfo{
				{Path: "/config/muxly", Prefix: "cfg"},
				{Path: "/Dev/muxly", Prefix: "dev"},
			},
			expected: map[string]string{
				"/config/muxly": "cfg/muxly",
				"/Dev/muxly":    "dev/muxly",
			},
		},
		{
			name: "three-way one alias",
			input: []models.PathInfo{
				{Path: "/config/muxly", Prefix: "cfg"},
				{Path: "/Dev/muxly"},
				{Path: "/Work/muxly"},
			},
			expected: map[string]string{
				"/config/muxly": "cfg/muxly",
				"/Dev/muxly":    "Dev/muxly",
				"/Work/muxly":   "Work/muxly",
			},
		},
		{
			name: "single dotdir no conflict",
			input: []models.PathInfo{
				{Path: "/Dev/.config"},
			},
			expected: map[string]string{
				"/Dev/.config": "config [dotdir]",
			},
		},
		{
			name: "dotdir and regular same basename",
			input: []models.PathInfo{
				{Path: "/Dev/.muxly"},
				{Path: "/Work/muxly"},
			},
			expected: map[string]string{
				"/Dev/.muxly": "muxly [dotdir]",
				"/Work/muxly": "muxly",
			},
		},
		{
			name: "two dotdirs different parents",
			input: []models.PathInfo{
				{Path: "/Dev/.config"},
				{Path: "/Work/.config"},
			},
			expected: map[string]string{
				"/Dev/.config":  "Dev/config [dotdir]",
				"/Work/.config": "Work/config [dotdir]",
			},
		},
		{
			name: "multiple leading dots",
			input: []models.PathInfo{
				{Path: "/Dev/..hidden"},
			},
			expected: map[string]string{
				"/Dev/..hidden": "hidden [dotdir x2]",
			},
		},
		{
			name: "triple dots",
			input: []models.PathInfo{
				{Path: "/Dev/...weird"},
			},
			expected: map[string]string{
				"/Dev/...weird": "weird [dotdir x3]",
			},
		},
		{
			name: "multi-dot conflict",
			input: []models.PathInfo{
				{Path: "/Dev/..foo"},
				{Path: "/Work/..foo"},
			},
			expected: map[string]string{
				"/Dev/..foo":  "Dev/foo [dotdir x2]",
				"/Work/..foo": "Work/foo [dotdir x2]",
			},
		},
		{
			name: "root level paths",
			input: []models.PathInfo{
				{Path: "/muxly"},
				{Path: "/other"},
			},
			expected: map[string]string{
				"/muxly": "muxly",
				"/other": "other",
			},
		},
		{
			name: "path with spaces",
			input: []models.PathInfo{
				{Path: "/My Projects/muxly"},
				{Path: "/Dev/muxly"},
			},
			expected: map[string]string{
				"/My Projects/muxly": "My Projects/muxly",
				"/Dev/muxly":         "Dev/muxly",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeduplicateDisplayNames(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("DeduplicateDisplayNames() returned %d entries, want %d", len(got), len(tt.expected))
			}
			for path, expectedName := range tt.expected {
				if gotName, ok := got[path]; !ok {
					t.Errorf("DeduplicateDisplayNames() missing path %q", path)
				} else if gotName != expectedName {
					t.Errorf("DeduplicateDisplayNames()[%q] = %q, want %q", path, gotName, expectedName)
				}
			}
		})
	}
}

func TestGetPathSuffix(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		depth    int
		expected string
	}{
		{"depth 1", "/home/user/Dev/muxly", 1, "muxly"},
		{"depth 2", "/home/user/Dev/muxly", 2, "Dev/muxly"},
		{"depth 3", "/home/user/Dev/muxly", 3, "user/Dev/muxly"},
		{"depth exceeds path", "/home/user/Dev/muxly", 10, "home/user/Dev/muxly"},
		{"short path depth 1", "/muxly", 1, "muxly"},
		{"short path depth 5", "/muxly", 5, "muxly"},
		{"relative path", "muxly", 1, "muxly"},
		{"empty path", "", 1, "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getPathSuffix(tt.path, tt.depth)
			if got != tt.expected {
				t.Errorf("getPathSuffix(%q, %d) = %q, want %q", tt.path, tt.depth, got, tt.expected)
			}
		})
	}
}
