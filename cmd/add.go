package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Pairadux/muxly/internal/models"
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
  muxly add entry ~/Dev/my-project
  muxly add e .
  muxly add scan ~/Dev --depth 2 --alias dev
  muxly add s ~/projects`,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

// resolveInputPath handles path resolution including special handling for relative paths like "." and ".."
func resolveInputPath(inputPath string) (string, error) {
	// Handle relative paths like "." and ".."
	// filepath.Abs converts them to absolute paths based on current working directory
	if inputPath == "." || inputPath == ".." {
		absPath, err := filepath.Abs(inputPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve relative path %q: %w", inputPath, err)
		}
		// Still pass through ResolvePath to handle any env vars and path cleaning
		return utility.ResolvePath(absPath)
	}

	// For absolute paths, tilde paths, or paths with env vars
	return utility.ResolvePath(inputPath)
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

		depth := scanDir.GetDepth(0, cfg.Settings.DefaultDepth)

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
