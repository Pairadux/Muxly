package utility

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFileAtomic replaces the contents of path in one step: the new contents go in a
// temporary file next to it, which is then renamed over it. A failure part way through
// leaves the old file as it was, where writing in place could leave half of each.
//
// If path is a symlink, as it often is for a file kept in a dotfiles repo, the file it
// points to is what's replaced, since renaming onto the link would turn it into a plain
// file. An existing file keeps its permissions, and perm is for a new one.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	target, err := filepath.EvalSymlinks(path)
	switch {
	case err == nil:
		info, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("checking %s: %w", target, err)
		}
		perm = info.Mode().Perm()
	case errors.Is(err, fs.ErrNotExist):
		target = path
	default:
		return fmt.Errorf("following %s: %w", path, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating a temporary file: %w", err)
	}
	// Once the rename has happened there's nothing left to remove, and if this fails
	// after an earlier error, that error is the one worth reporting.
	defer func() { _ = os.Remove(tmp.Name()) }()

	_, err = tmp.Write(data)
	err = errors.Join(err, tmp.Chmod(perm), tmp.Sync(), tmp.Close())
	if err != nil {
		return fmt.Errorf("writing %s: %w", tmp.Name(), err)
	}
	if err := os.Rename(tmp.Name(), target); err != nil {
		return fmt.Errorf("replacing %s: %w", target, err)
	}
	return nil
}
