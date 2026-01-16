package utility

// IMPORTS {{{
import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Pairadux/muxly/internal/constants"
	"github.com/charlievieth/fastwalk"
	"github.com/mitchellh/go-homedir"
) // }}}

// ResolvePath takes an "unknown" path pattern and returns an absolute path.
//
//   - Absolute (/…):               returned as-is
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
		return p, nil
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

// GetSubDirs returns all subdirectories within root, up to maxDepth levels deep.
//
// Depth examples (assuming root = "/home/user/Dev"):
//
//	maxDepth = 1: /home/user/Dev/project1, /home/user/Dev/project2
//	maxDepth = 2: above + /home/user/Dev/project1/src, /home/user/Dev/project2/src
//
// Uses fastwalk for efficient concurrent traversal. The root directory itself is
// always excluded from results. Individual path errors are logged to stderr but
// don't stop the scan or cause an error return.
//
// Performance: Results are collected concurrently via a buffered channel
// (size: constants.DefaultChannelBufferSize).
func GetSubDirs(maxDepth int, root string) ([]string, error) {
	// PERF: Channel buffer size may be too small for large directory trees, consider making it configurable
	dirChan := make(chan string, constants.DefaultChannelBufferSize)
	cfg := &fastwalk.Config{MaxDepth: maxDepth}
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Walk error %q: %v\n", path, err)
			return nil
		}
		// IDEA: might make this into a flag or config option
		// ExcludeRootDir
		if d.Name() == filepath.Base(root) {
			return nil
		}
		if d.IsDir() {
			dirChan <- path
		}
		return nil
	}
	// PERF: Pre-allocate dirs slice with estimated capacity to reduce allocations
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
