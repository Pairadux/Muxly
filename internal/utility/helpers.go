// Package utility provides path resolution and directory walking.
package utility

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/charlievieth/fastwalk"
	"github.com/mitchellh/go-homedir"
)

// ResolvePath takes an "unknown" path pattern and returns a clean absolute path.
//
//   - Absolute (/…):               used as it is
//   - Home (~ or ~/…):             expanded via os.UserHomeDir()
//   - Explicit relative (./, ../): error
//
// Automatically removes unnecessary escape sequences (like \  for spaces)
// since Go's exec.Command handles spaces properly without escaping.
func ResolvePath(p string) (string, error) {
	// Remove unnecessary escape sequences that users might add
	p = strings.ReplaceAll(p, "\\ ", " ")

	p = os.ExpandEnv(p)

	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	if strings.HasPrefix(p, "~") {
		return homedir.Expand(p)
	}
	if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") || p == "." || p == ".." {
		return "", fmt.Errorf("relative paths (%q) are not allowed", p)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(home, p)), nil
}

// IsExplicitPath reports whether a command line argument can only be a path, and not a
// name: ".", "..", or anything starting with "/", "./", "../" or "~/".
func IsExplicitPath(arg string) bool {
	if arg == "." || arg == ".." || arg == "~" {
		return true
	}
	for _, prefix := range []string{"/", "./", "../", "~/"} {
		if strings.HasPrefix(arg, prefix) {
			return true
		}
	}
	return false
}

// ResolveArgumentPath returns the clean absolute path for a path typed on the command
// line. Unlike a path in the config, a relative one starts from the current folder, as it
// would in a shell. The shell has also already expanded it, so a "$" in it is a real "$".
func ResolveArgumentPath(arg string) (string, error) {
	if arg == "~" || strings.HasPrefix(arg, "~/") {
		return homedir.Expand(arg)
	}
	return filepath.Abs(arg)
}

// IsWithin reports whether path is one of paths, or inside one of them.
func IsWithin(path string, paths models.StringSet) bool {
	for {
		if _, found := paths[path]; found {
			return true
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}

// RequireDirectory returns an error saying what's wrong unless path is a directory.
func RequireDirectory(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s does not exist", path)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

// GetSubDirs returns all subdirectories within root, up to maxDepth levels deep.
//
// Depth examples (assuming root = "/home/user/Dev"):
//
//	maxDepth = 1: /home/user/Dev/project1, /home/user/Dev/project2
//	maxDepth = 2: above + /home/user/Dev/project1/src, /home/user/Dev/project2/src
//
// Directories whose base names appear in ignoreDirNames, or whose paths appear in
// ignorePaths, are skipped entirely (not descended into), providing both correct
// filtering and a performance benefit.
//
// Uses fastwalk for efficient concurrent traversal. The root directory itself is always
// excluded from results. A path that can't be read doesn't stop the scan: it's returned
// with the others in problems, for the caller to report when it can.
func GetSubDirs(maxDepth int, root string, ignoreDirNames, ignorePaths models.StringSet) (dirs []string, problems []error) {
	var mu sync.Mutex
	cfg := &fastwalk.Config{MaxDepth: maxDepth}
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			mu.Lock()
			defer mu.Unlock()
			problems = append(problems, err)
			return nil
		}
		if path == root {
			return nil
		}
		if d.IsDir() {
			if _, ignored := ignoreDirNames[d.Name()]; ignored {
				return fastwalk.SkipDir
			}
			if _, ignored := ignorePaths[path]; ignored {
				return fastwalk.SkipDir
			}
			mu.Lock()
			defer mu.Unlock()
			dirs = append(dirs, path)
		}
		return nil
	}
	if err := fastwalk.Walk(cfg, root, walkFn); err != nil {
		problems = append(problems, err)
	}
	return dirs, problems
}
