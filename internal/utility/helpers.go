// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package utility

// IMPORTS {{{
import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Pairadux/tms/internal/models"
	"github.com/charlievieth/fastwalk"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
) // }}}

// ResolvePath takes an “unknown” path pattern and returns an absolute path.
//
//   - Absolute (/…):               returned as-is
//   - Home (~ or ~/…):             expanded via os.UserHomeDir()
//   - Explicit relative (./, ../): error
func ResolvePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	if strings.HasPrefix(p, "~") {
		return homedir.Expand(p)
	}
	if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") || p == "." || p == ".." {
		return "", errors.New("relative paths not allowed: " + p)
	}
	return "", errors.New("path type not supported: '" + p + "'")
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

func WarnOnConfigIssues(cfg *models.Config) {
	if cfg.Editor == "" {
		fmt.Fprintln(os.Stderr, "Warning: editor not set, defaulting to 'vi'")
	}

	if cfg.FallbackSession.Name == "" {
		fmt.Fprintln(os.Stderr, "fallback_session.name is missing, defaulting to 'Default'")
	}

	if cfg.FallbackSession.Path == "" {
		fmt.Fprintln(os.Stderr, "fallback_session.path is missing, defaulting to '~/'")
	}

	if len(cfg.FallbackSession.Layout.Windows) == 0 {
		fmt.Fprintln(os.Stderr, "fallback_session.layout.windows is empty, using default layout")
	}
}

func VerifyExternalUtils() error {
	var missing []string

	if _, err := exec.LookPath("tmux"); err != nil {
		missing = append(missing, "tmux")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		missing = append(missing, "fzf")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}

	return nil
}
