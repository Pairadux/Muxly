// Package selector works out what the picker offers: it finds the configured directories,
// gives each one a label, and matches them with the running tmux sessions.
package selector

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/utility"
)

// Directory is a configured directory. Its path is what identifies it, because a label
// depends on what other directories exist, and a session's name can be anything.
type Directory struct {
	// Path is absolute and clean, and appears once however often it's configured.
	Path string
	// Label is what the picker calls the directory, and no other directory has it.
	Label string
	// Alias comes from the scan directory it was found in, if that has one.
	Alias string
	// Template names the template for new sessions here, if one is set.
	Template string
}

// Directories finds everything in scan_dirs and entry_dirs that isn't ignored, sorted by
// path. A directory in both takes its template from entry_dirs, as the more specific.
//
// A flagDepth above zero overrides the depth of every scan directory.
func Directories(cfg *models.Config, flagDepth int, verbose bool) []Directory {
	ignorePaths, ignoreNames := buildIgnoreSets(cfg)
	found := make(map[string]*Directory)

	for _, scanDir := range cfg.ScanDirs {
		for _, path := range scan(cfg, scanDir, flagDepth, ignorePaths, ignoreNames, verbose) {
			if _, seen := found[path]; !seen {
				found[path] = &Directory{Path: path, Alias: scanDir.Alias, Template: scanDir.Template}
			}
		}
	}

	for _, entryDir := range cfg.EntryDirs {
		path, err := utility.ResolvePath(entryDir.Path)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to resolve entry directory %s: %v\n", entryDir.Path, err)
			}
			continue
		}
		if utility.IsWithin(path, ignorePaths) {
			continue
		}
		dir, seen := found[path]
		if !seen {
			dir = &Directory{Path: path}
			found[path] = dir
		}
		if entryDir.Template != "" {
			dir.Template = entryDir.Template
		}
	}

	dirs := make([]Directory, 0, len(found))
	for _, dir := range found {
		dirs = append(dirs, *dir)
	}
	slices.SortFunc(dirs, func(a, b Directory) int { return strings.Compare(a.Path, b.Path) })
	assignLabels(dirs)
	return dirs
}

// buildIgnoreSets partitions ignore directories into two sets for O(1) lookup:
//   - ignorePaths: resolved absolute paths for entries that look like paths
//     (contain "/" or start with "~"), e.g. "~/projects/archived"
//   - ignoreNames: bare directory names matched against basenames during scanning,
//     e.g. ".git", "node_modules"
//
// Base directories (config.BaseIgnoreDirs) are always included and cannot be
// overridden. User-configured ignore_dirs entries are additive on top of these.
//
// This allows ignore_dirs to support both styles:
//
//	ignore_dirs:
//	  - target            # bare name  — matches any directory named "target" at any depth
//	  - ~/projects/old    # path       — matches that resolved directory and all inside it
func buildIgnoreSets(cfg *models.Config) (ignorePaths, ignoreNames models.StringSet) {
	ignorePaths = make(models.StringSet)
	ignoreNames = make(models.StringSet)

	for _, dir := range config.BaseIgnoreDirs {
		ignoreNames[dir] = struct{}{}
	}

	for _, dir := range cfg.IgnoreDirs {
		if strings.Contains(dir, "/") || strings.HasPrefix(dir, "~") {
			if resolved, err := utility.ResolvePath(dir); err == nil {
				ignorePaths[resolved] = struct{}{}
			}
		} else {
			ignoreNames[dir] = struct{}{}
		}
	}
	return ignorePaths, ignoreNames
}

// scan returns the subdirectories of one scan directory.
func scan(cfg *models.Config, scanDir models.ScanDir, flagDepth int, ignorePaths, ignoreNames models.StringSet, verbose bool) []string {
	root, err := utility.ResolvePath(scanDir.Path)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to resolve scan directory %s: %v\n", scanDir.Path, err)
		}
		return nil
	}
	if utility.IsWithin(root, ignorePaths) {
		return nil
	}

	depth := scanDir.GetDepth(flagDepth, cfg.Settings.DefaultDepth)
	subDirs, err := utility.GetSubDirs(depth, root, ignoreNames, ignorePaths)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", root, err)
		}
		return nil
	}
	return subDirs
}
