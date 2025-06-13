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

	"github.com/Pairadux/tms/internal/models"
	"github.com/charlievieth/fastwalk"
	"github.com/spf13/viper"
	"github.com/mitchellh/go-homedir"
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
		// NOTE: might make this into a flag or config option
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

// validateConfig ensures that the application configuration is valid and complete.
// It checks for the presence of a config file and verifies that at least one
// directory is configured for scanning (either scan_dirs or entry_dirs).
// Returns an error with helpful instructions if validation fails.
func ValidateConfig(cfg *models.Config) error {
	// FIXME: make this check the values of the setup config struct to ensure compliance
	if viper.ConfigFileUsed() == "" {
		return fmt.Errorf("no config file found\nRun 'tms config init' to create one, or use --config to specify a path")
	}
	if (len(cfg.ScanDirs) == 0) && (len(cfg.EntryDirs) == 0) {
		return fmt.Errorf("no directories configured for scanning")
	}

	return nil
}

// TODO: create this
func VerifyExternalUtils() error {
	// tmux
	// fzf
	return nil
}
