package utility

import (
	"os"
	"path/filepath"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names
}

func TestWriteFileAtomicCreatesAFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.yaml")

	if err := WriteFileAtomic(path, []byte("new"), 0o640); err != nil {
		t.Fatal(err)
	}

	if got := readFile(t, path); got != "new" {
		t.Errorf("contents = %q", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %o", info.Mode().Perm())
	}
	if names := dirEntries(t, dir); len(names) != 1 {
		t.Errorf("left behind: %q", names)
	}
}

func TestWriteFileAtomicKeepsAnExistingFilesMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(path, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o", info.Mode().Perm())
	}
	if got := readFile(t, path); got != "new" {
		t.Errorf("contents = %q", got)
	}
}

func TestWriteFileAtomicFollowsASymlink(t *testing.T) {
	repo := t.TempDir()
	target := filepath.Join(repo, "dot_config.yaml")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(link, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink became a plain file")
	}
	if got := readFile(t, target); got != "new" {
		t.Errorf("target contents = %q", got)
	}
	if names := dirEntries(t, repo); len(names) != 1 {
		t.Errorf("left behind: %q", names)
	}
}

func TestWriteFileAtomicFailureKeepsTheOldFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root can write anywhere")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dir, 0o700); err != nil {
			t.Error(err)
		}
	})

	if err := WriteFileAtomic(path, []byte("new"), 0o644); err == nil {
		t.Error("want an error")
	}

	if got := readFile(t, path); got != "old" {
		t.Errorf("contents = %q", got)
	}
}

func TestWriteFileAtomicCleansUpAfterAFailure(t *testing.T) {
	parent := t.TempDir()
	// Renaming a file onto a directory fails, and only after the temporary file exists.
	target := filepath.Join(parent, "a directory")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := WriteFileAtomic(target, []byte("new"), 0o644); err == nil {
		t.Error("want an error")
	}

	if names := dirEntries(t, parent); len(names) != 1 {
		t.Errorf("left behind: %q", names)
	}
}
