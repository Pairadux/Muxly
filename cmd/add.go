// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package cmd

// IMPORTS {{{
import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/utility"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
) // }}}

// addCmd represents the add command
var addCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Add a directory to entry_dirs",
	Long: `Add a directory to entry_dirs in the configuration file.

The path can be absolute, relative, or use tilde expansion.
Relative paths (like . or ..) will be converted to absolute paths.

Examples:
  muxly add .                    # Add current directory
  muxly add ~/Dev/my-project     # Add a specific project
  muxly add /path/to/directory   # Add absolute path`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		inputPath := args[0]

		var resolvedPath string
		var err error

		// Handle relative paths like "." and ".."
		// filepath.Abs converts them to absolute paths based on current working directory
		if inputPath == "." || inputPath == ".." {
			absPath, err := filepath.Abs(inputPath)
			if err != nil {
				return fmt.Errorf("failed to resolve relative path %q: %w", inputPath, err)
			}
			// Still pass through ResolvePath to handle any env vars and path cleaning
			resolvedPath, err = utility.ResolvePath(absPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path %q: %w", absPath, err)
			}
		} else {
			// For absolute paths, tilde paths, or paths with env vars
			resolvedPath, err = utility.ResolvePath(inputPath)
			if err != nil {
				return fmt.Errorf("failed to resolve path %q: %w", inputPath, err)
			}
		}

		// Verify the path actually exists on the filesystem
		if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", resolvedPath)
		} else if err != nil {
			return fmt.Errorf("failed to access path %q: %w", resolvedPath, err)
		}

		// Check if this path would already be discovered by a scan_dir
		// This prevents redundant configuration
		if matchedScanDir, depth, found := wouldBeFoundByScanDirs(resolvedPath, cfg.ScanDirs); found {
			return fmt.Errorf("path %q would already be found by scan_dir %s (depth: %d)\nNo need to add it to entry_dirs", resolvedPath, matchedScanDir.String(), depth)
		}

		// Check if already in entry_dirs to avoid duplicates
		if slices.Contains(cfg.EntryDirs, resolvedPath) {
			fmt.Printf("Path %q is already in entry_dirs\n", resolvedPath)
			return nil
		}

		// Add to entry_dirs and write config using viper
		updatedEntryDirs := append(cfg.EntryDirs, resolvedPath)
		viper.Set("entry_dirs", updatedEntryDirs)

		if err := viper.WriteConfig(); err != nil {
			return fmt.Errorf("failed to write config: %w", err)
		}

		fmt.Printf("Added %q to entry_dirs\n", resolvedPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
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

		depth := scanDir.GetDepth(0, cfg.DefaultDepth)

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
//   Given scan_dir: {path: /home/user/Dev, depth: 2}
//
//   isWithinDepth(/home/user/Dev/project1, /home/user/Dev, 2)
//     -> relative path: "project1" (depth 1) -> true
//
//   isWithinDepth(/home/user/Dev/project1/src, /home/user/Dev, 2)
//     -> relative path: "project1/src" (depth 2) -> true
//
//   isWithinDepth(/home/user/Dev/project1/src/main, /home/user/Dev, 2)
//     -> relative path: "project1/src/main" (depth 3) -> false
//
//   isWithinDepth(/home/user/Dev, /home/user/Dev, 2)
//     -> relative path: "." (scan dir itself) -> false
//
//   isWithinDepth(/different/path, /home/user/Dev, 2)
//     -> not under scan path -> false
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
