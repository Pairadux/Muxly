package config

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/state"
	"github.com/Pairadux/muxly/internal/utility"

	"github.com/spf13/cobra"
)

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add directories to configuration",
	Long: `Add directories to entry_dirs or scan_dirs in the configuration file.

Use subcommands to specify where to add the directory:
  entry (e) - Add to entry_dirs (direct entries, not scanned)
  scan (s)  - Add to scan_dirs (scanned recursively with depth)

Examples:
  muxly config add entry ~/Dev/my-project
  muxly config add e .
  muxly config add scan ~/Dev --depth 2 --alias dev
  muxly config add s ~/projects`,
}

func init() {
	Cmd.AddCommand(addCmd)
}

// resolvesTo reports whether a path as written in the config means the given clean,
// absolute path.
func resolvesTo(configured, resolved string) bool {
	existing, err := utility.ResolvePath(configured)
	return err == nil && existing == resolved
}

// findConfigured returns the position of the item a path on the command line means, or -1.
// What the path resolves to comes first, as it does when adding. Failing that, it can be
// exactly as the item is written, which is how "muxly config list" shows it, and the only
// way to name one that doesn't resolve, or whose bare path is measured from somewhere else.
func findConfigured[T any](items []T, path func(T) string, arg, resolved string) int {
	if i := slices.IndexFunc(items, func(item T) bool { return resolvesTo(path(item), resolved) }); i != -1 {
		return i
	}
	return slices.IndexFunc(items, func(item T) bool { return path(item) == arg })
}

// pathForConfig returns how to write a directory in the config file. Paths there are read
// differently from those on the command line, because a "$" in one expands, so it makes
// sure this one will still mean the same directory when it's read back.
func pathForConfig(resolved string) (string, error) {
	written := utility.TildePath(resolved)
	if !resolvesTo(written, resolved) {
		return "", fmt.Errorf("%s can't be listed in the config, because $ and \\ in paths there don't mean themselves", resolved)
	}
	return written, nil
}

// wouldBeFoundByScanDirs checks if a target path would be discovered by any configured scan_dir.
//
// Performance: O(m) where m is the number of scan_dirs. Uses O(1) path arithmetic
// instead of O(n) filesystem scanning.
//
// Returns the matching ScanDir, its effective depth, and true if found.
func wouldBeFoundByScanDirs(targetPath string, scanDirs []models.ScanDir) (*models.ScanDir, int, bool) {
	for _, scanDir := range scanDirs {
		resolvedScanPath, err := utility.ResolvePath(scanDir.Path)
		if err != nil {
			continue
		}

		depth := scanDir.GetDepth(0, state.Cfg.Settings.DefaultDepth)

		if isWithinDepth(targetPath, resolvedScanPath, depth) {
			return &scanDir, depth, true
		}
	}
	return nil, 0, false
}

// isWithinDepth checks if targetPath would be found when scanning scanPath at the given depth.
//
// Uses O(1) path arithmetic instead of O(n) filesystem scanning.
//
// Examples:
//
//	Given scan_dir: {path: /home/user/Dev, depth: 2}
//
//	isWithinDepth(/home/user/Dev/project1, /home/user/Dev, 2)
//	  -> relative path: "project1" (depth 1) -> true
//
//	isWithinDepth(/home/user/Dev/project1/src, /home/user/Dev, 2)
//	  -> relative path: "project1/src" (depth 2) -> true
//
//	isWithinDepth(/home/user/Dev/project1/src/main, /home/user/Dev, 2)
//	  -> relative path: "project1/src/main" (depth 3) -> false
//
//	isWithinDepth(/home/user/Dev, /home/user/Dev, 2)
//	  -> relative path: "." (scan dir itself) -> false
//
//	isWithinDepth(/different/path, /home/user/Dev, 2)
//	  -> not under scan path -> false
func isWithinDepth(targetPath, scanPath string, depth int) bool {
	relPath, err := filepath.Rel(scanPath, targetPath)
	if err != nil || strings.HasPrefix(relPath, "..") {
		return false
	}

	if relPath == "." {
		return false
	}

	pathDepth := strings.Count(relPath, string(filepath.Separator)) + 1
	return pathDepth <= depth
}
