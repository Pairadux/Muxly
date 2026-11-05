package utility

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

func TestResolvePath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
		errContains string
	}{
		{
			name:     "absolute path returned as-is",
			input:    "/usr/local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "trailing slash removed",
			input:    "/usr/local/bin/",
			expected: "/usr/local/bin",
		},
		{
			name:     "dot segments removed",
			input:    "/usr/./local/../local/bin",
			expected: "/usr/local/bin",
		},
		{
			name:     "home tilde with a trailing slash",
			input:    "~/Dev/",
			expected: filepath.Join(home, "Dev"),
		},
		{
			name:     "home tilde expands",
			input:    "~",
			expected: home,
		},
		{
			name:     "home tilde with path",
			input:    "~/Dev/project",
			expected: filepath.Join(home, "Dev/project"),
		},
		{
			name:        "relative dot path errors",
			input:       "./config",
			expectError: true,
			errContains: "relative paths",
		},
		{
			name:        "relative double dot path errors",
			input:       "../config",
			expectError: true,
			errContains: "relative paths",
		},
		{
			name:        "single dot errors",
			input:       ".",
			expectError: true,
			errContains: "relative paths",
		},
		{
			name:        "double dot errors",
			input:       "..",
			expectError: true,
			errContains: "relative paths",
		},
		{
			name:     "escaped spaces are unescaped",
			input:    "/path/to/my\\ project",
			expected: "/path/to/my project",
		},
		{
			name:     "bare name resolves to home",
			input:    "Documents",
			expected: filepath.Join(home, "Documents"),
		},
		{
			name:     "bare path with slash resolves to home",
			input:    "Dev/project",
			expected: filepath.Join(home, "Dev/project"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolvePath(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("ResolvePath(%q) expected error containing %q, got nil", tt.input, tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ResolvePath(%q) error = %q, want error containing %q", tt.input, err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("ResolvePath(%q) unexpected error: %v", tt.input, err)
				return
			}

			if got != tt.expected {
				t.Errorf("ResolvePath(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestResolvePathEnvExpansion(t *testing.T) {
	testVar := "MUXLY_TEST_VAR"
	testValue := "/custom/path"
	t.Setenv(testVar, testValue)

	input := "$" + testVar + "/subdir"
	expected := testValue + "/subdir"

	got, err := ResolvePath(input)
	if err != nil {
		t.Fatalf("ResolvePath(%q) unexpected error: %v", input, err)
	}

	if got != expected {
		t.Errorf("ResolvePath(%q) = %q, want %q", input, got, expected)
	}
}

func TestGetSubDirs(t *testing.T) {
	tempDir := t.TempDir()

	mustMkdirAll(t, filepath.Join(tempDir, "project1"))
	mustMkdirAll(t, filepath.Join(tempDir, "project2"))
	mustMkdirAll(t, filepath.Join(tempDir, "project1", "src"))
	mustMkdirAll(t, filepath.Join(tempDir, "project2", "src", "lib"))
	mustWriteFile(t, filepath.Join(tempDir, "file.txt"))

	tests := []struct {
		name          string
		maxDepth      int
		minExpected   int
		shouldContain []string
		shouldExclude []string
	}{
		{
			name:        "depth 1 gets immediate subdirs",
			maxDepth:    1,
			minExpected: 2,
			shouldContain: []string{
				filepath.Join(tempDir, "project1"),
				filepath.Join(tempDir, "project2"),
			},
			shouldExclude: []string{
				filepath.Join(tempDir, "project1", "src"),
			},
		},
		{
			name:        "depth 2 gets nested subdirs",
			maxDepth:    2,
			minExpected: 4,
			shouldContain: []string{
				filepath.Join(tempDir, "project1"),
				filepath.Join(tempDir, "project1", "src"),
				filepath.Join(tempDir, "project2", "src"),
			},
		},
		{
			name:        "depth 3 gets deeply nested",
			maxDepth:    3,
			minExpected: 5,
			shouldContain: []string{
				filepath.Join(tempDir, "project2", "src", "lib"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirs, problems := GetSubDirs(tt.maxDepth, tempDir, nil, nil)
			if len(problems) > 0 {
				t.Fatalf("GetSubDirs(%d, %q) unexpected problems: %v", tt.maxDepth, tempDir, problems)
			}

			if len(dirs) < tt.minExpected {
				t.Errorf("GetSubDirs(%d, %q) returned %d dirs, want at least %d", tt.maxDepth, tempDir, len(dirs), tt.minExpected)
			}

			dirSet := make(map[string]bool)
			for _, d := range dirs {
				dirSet[d] = true
			}

			for _, expected := range tt.shouldContain {
				if !dirSet[expected] {
					t.Errorf("GetSubDirs(%d, %q) should contain %q", tt.maxDepth, tempDir, expected)
				}
			}

			for _, excluded := range tt.shouldExclude {
				if dirSet[excluded] {
					t.Errorf("GetSubDirs(%d, %q) should not contain %q", tt.maxDepth, tempDir, excluded)
				}
			}

			if dirSet[tempDir] {
				t.Errorf("GetSubDirs should exclude root directory %q", tempDir)
			}
		})
	}
}

func TestGetSubDirsExcludesFiles(t *testing.T) {
	tempDir := t.TempDir()

	mustMkdirAll(t, filepath.Join(tempDir, "dir1"))
	mustWriteFile(t, filepath.Join(tempDir, "file1.txt"))
	mustWriteFile(t, filepath.Join(tempDir, "file2.go"))

	dirs, problems := GetSubDirs(1, tempDir, nil, nil)
	if len(problems) > 0 {
		t.Fatalf("GetSubDirs unexpected problems: %v", problems)
	}

	for _, d := range dirs {
		if strings.Contains(d, "file") {
			t.Errorf("GetSubDirs should not include files, found: %q", d)
		}
	}
}

func TestGetSubDirsEmptyDir(t *testing.T) {
	tempDir := t.TempDir()

	dirs, problems := GetSubDirs(1, tempDir, nil, nil)
	if len(problems) > 0 {
		t.Fatalf("GetSubDirs unexpected problems: %v", problems)
	}

	if len(dirs) != 0 {
		t.Errorf("GetSubDirs on empty dir should return empty slice, got %d dirs", len(dirs))
	}
}

func TestGetSubDirsIgnoreNames(t *testing.T) {
	tempDir := t.TempDir()

	mustMkdirAll(t, filepath.Join(tempDir, "project1"))
	mustMkdirAll(t, filepath.Join(tempDir, "project1", ".git"))
	mustMkdirAll(t, filepath.Join(tempDir, "project1", ".git", "objects"))
	mustMkdirAll(t, filepath.Join(tempDir, "project1", "node_modules"))
	mustMkdirAll(t, filepath.Join(tempDir, "project1", "node_modules", "lodash"))
	mustMkdirAll(t, filepath.Join(tempDir, "project1", "src"))
	mustMkdirAll(t, filepath.Join(tempDir, "project2"))

	ignoreNames := models.StringSet{
		".git":         {},
		"node_modules": {},
	}

	dirs, problems := GetSubDirs(3, tempDir, ignoreNames, nil)
	if len(problems) > 0 {
		t.Fatalf("GetSubDirs unexpected problems: %v", problems)
	}

	dirSet := make(map[string]bool)
	for _, d := range dirs {
		dirSet[d] = true
	}

	// Should include non-ignored dirs
	for _, expected := range []string{
		filepath.Join(tempDir, "project1"),
		filepath.Join(tempDir, "project1", "src"),
		filepath.Join(tempDir, "project2"),
	} {
		if !dirSet[expected] {
			t.Errorf("GetSubDirs should contain %q", expected)
		}
	}

	// Should exclude ignored dirs and their children
	for _, excluded := range []string{
		filepath.Join(tempDir, "project1", ".git"),
		filepath.Join(tempDir, "project1", ".git", "objects"),
		filepath.Join(tempDir, "project1", "node_modules"),
		filepath.Join(tempDir, "project1", "node_modules", "lodash"),
	} {
		if dirSet[excluded] {
			t.Errorf("GetSubDirs should not contain ignored dir %q", excluded)
		}
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Only the root itself is left out, not everything that shares its name.
func TestGetSubDirsListsDirectoriesNamedLikeTheRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "scan")
	child := filepath.Join(root, "scan")
	nested := filepath.Join(root, "nested", "scan")
	mustMkdirAll(t, child)
	mustMkdirAll(t, nested)

	dirs, problems := GetSubDirs(2, root, nil, nil)
	if len(problems) > 0 {
		t.Fatalf("GetSubDirs unexpected problems: %v", problems)
	}

	for _, want := range []string{child, nested} {
		if !slices.Contains(dirs, want) {
			t.Errorf("GetSubDirs = %q, want it to contain %q", dirs, want)
		}
	}
	if slices.Contains(dirs, root) {
		t.Errorf("GetSubDirs = %q, want it to leave out the root", dirs)
	}
}

func TestGetSubDirsIgnorePaths(t *testing.T) {
	root := t.TempDir()
	old := filepath.Join(root, "old")
	child := filepath.Join(old, "child")
	keep := filepath.Join(root, "keep")
	sameName := filepath.Join(keep, "old")
	mustMkdirAll(t, child)
	mustMkdirAll(t, sameName)

	dirs, problems := GetSubDirs(3, root, nil, models.StringSet{old: {}})
	if len(problems) > 0 {
		t.Fatalf("GetSubDirs unexpected problems: %v", problems)
	}

	for _, ignored := range []string{old, child} {
		if slices.Contains(dirs, ignored) {
			t.Errorf("GetSubDirs = %q, want it to leave out %q", dirs, ignored)
		}
	}
	for _, want := range []string{keep, sameName} {
		if !slices.Contains(dirs, want) {
			t.Errorf("GetSubDirs = %q, want it to contain %q", dirs, want)
		}
	}
}

func TestGetSubDirsMissingRootIsAProblem(t *testing.T) {
	dirs, problems := GetSubDirs(1, filepath.Join(t.TempDir(), "gone"), nil, nil)

	if len(dirs) != 0 || len(problems) != 1 {
		t.Errorf("GetSubDirs = %q, %v, want no dirs and one problem", dirs, problems)
	}
}
