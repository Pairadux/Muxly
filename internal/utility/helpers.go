// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package utility

// IMPORTS {{{
import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Pairadux/Tmux-Sessionizer/internal/constants"
	"github.com/charlievieth/fastwalk"
	"github.com/mitchellh/go-homedir"
) // }}}

// ResolvePath takes an “unknown” path pattern and returns an absolute path.
//
//   - Absolute (/…):               returned as-is
//   - Home (~ or ~/…):             expanded via os.UserHomeDir()
//   - Explicit relative (./, ../): error
func ResolvePath(p string) (string, error) {
	// IDEA: I would like to accept Base paths (Documents for ~/Documents) and ENV variables
	if filepath.IsAbs(p) {
		return p, nil
	}
	if strings.HasPrefix(p, "~") {
		return homedir.Expand(p)
	}
	if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") || p == "." || p == ".." {
		return homedir.Dir()
	}
	return homedir.Dir()
}

// GetSubDirs returns all subdirectories within the specified root directory,
// up to the given maximum depth. It uses fastwalk for efficient directory
// traversal and processes directories concurrently via a goroutine and channel.
//
// Walk errors for individual paths are printed to stderr but do not stop
// the traversal or cause the function to return an error.
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
