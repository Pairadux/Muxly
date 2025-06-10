// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package utility

// IMPORTS {{{
import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/charlievieth/fastwalk"
) // }}}

// ResolvePath takes an “unknown” path pattern and returns an absolute path.
//
//   - Absolute (/…):               returned as-is
//   - Home (~ or ~/…):             expanded via os.UserHomeDir()
//   - Explicit relative (./, ../): error
//   - Implicit (foo/bar):          treated as ~/foo/bar
func ResolvePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		switch {
		case p == "~":
			return home, nil
		case strings.HasPrefix(p, "~/"):
			return filepath.Join(home, p[2:]), nil
		default:
			return "", errors.New("invalid home-path syntax")
		}
	}
	if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") || p == "." || p == ".." {
		return "", errors.New("relative paths not allowed: " + p)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, p), nil
}

// GetSubDirs returns all subdirectories within the specified root directory,
// up to the given maximum depth. It uses fastwalk for efficient directory
// traversal and processes directories concurrently via a goroutine and channel.
//
// Walk errors for individual paths are printed to stderr but do not stop
// the traversal or cause the function to return an error.
func GetSubDirs(maxDepth int, root string) ([]string, error) {
	dirChan := make(chan string, 100)
	cfg := &fastwalk.Config{MaxDepth: maxDepth}
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "walk error %q: %v\n", path, err)
			return nil
		}
		if d.IsDir() {
			dirChan <- path
		}
		return nil
	}
	var dirs []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		for dir := range dirChan {
			dirs = append(dirs, dir)
		}
	}()
	err := fastwalk.Walk(cfg, root, walkFn)
	close(dirChan)
	<-done
	if err != nil {
		return nil, err
	}
	return dirs, nil
}

