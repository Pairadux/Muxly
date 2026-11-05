// Package selector works out what the picker offers: it finds the configured directories,
// gives each one a label, and matches them with the running tmux sessions.
package selector

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/tmux"
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
	// Missing means it's in entry_dirs, but isn't a directory that exists.
	Missing bool
}

// Directories finds everything in scan_dirs and entry_dirs that isn't ignored, sorted by
// path, along with warnings about anything it couldn't use. A directory in both takes its
// template from entry_dirs, as the more specific.
//
// It prints nothing, because the picker may be about to take over the terminal. The caller
// reports the warnings when it's safe to.
//
// A flagDepth above zero overrides the depth of every scan directory.
func Directories(cfg *models.Config, flagDepth int) ([]Directory, []string) {
	ignorePaths, ignoreNames, warnings := buildIgnoreSets(cfg)
	found := make(map[string]*Directory)

	// tmux lists a session on two lines if its path has a newline, so muxly could open a
	// session there but never find it again.
	usable := func(path string) bool {
		if strings.Contains(path, "\n") {
			warnings = append(warnings, fmt.Sprintf("skipping %q: a path with a newline in it isn't supported", path))
			return false
		}
		return true
	}

	for _, scanDir := range cfg.ScanDirs {
		paths, scanWarnings := scan(cfg, scanDir, flagDepth, ignorePaths, ignoreNames)
		warnings = append(warnings, scanWarnings...)
		for _, path := range paths {
			if _, seen := found[path]; !seen && usable(path) {
				found[path] = &Directory{Path: path, Alias: scanDir.Alias, Template: scanDir.Template}
			}
		}
	}

	for _, entryDir := range cfg.EntryDirs {
		path, err := utility.ResolvePath(entryDir.Path)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("can't use entry_dirs path %s: %v", entryDir.Path, err))
			continue
		}
		if utility.IsWithin(path, ignorePaths) || !usable(path) {
			continue
		}
		dir, seen := found[path]
		if !seen {
			dir = &Directory{Path: path, Missing: utility.RequireDirectory(path) != nil}
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
	return dirs, warnings
}

// Find returns the directories a typed name could mean. A directory with exactly that
// label is the only match. Otherwise a name matches a label or folder name if both make
// the same session name, so case and symbols don't matter: "myproject" finds "MyProject",
// "config" finds ".config" and "dev-src" finds "Dev/src".
func Find(dirs []Directory, name string) []Directory {
	for _, dir := range dirs {
		if dir.Label == name {
			return []Directory{dir}
		}
	}

	wanted := tmux.SessionName(name)
	var matches []Directory
	for _, dir := range dirs {
		if wanted == tmux.SessionName(dir.Label) || wanted == tmux.SessionName(filepath.Base(dir.Path)) {
			matches = append(matches, dir)
		}
	}
	return matches
}

// FindPath returns the directory with the given clean, absolute path.
func FindPath(dirs []Directory, path string) (Directory, bool) {
	i := slices.IndexFunc(dirs, func(dir Directory) bool { return dir.Path == path })
	if i < 0 {
		return Directory{}, false
	}
	return dirs[i], true
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
func buildIgnoreSets(cfg *models.Config) (ignorePaths, ignoreNames models.StringSet, warnings []string) {
	ignorePaths = make(models.StringSet)
	ignoreNames = make(models.StringSet)

	for _, dir := range config.BaseIgnoreDirs {
		ignoreNames[dir] = struct{}{}
	}

	for _, dir := range cfg.IgnoreDirs {
		if strings.Contains(dir, "/") || strings.HasPrefix(dir, "~") {
			resolved, err := utility.ResolvePath(dir)
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("can't use ignore_dirs path %s: %v", dir, err))
				continue
			}
			ignorePaths[resolved] = struct{}{}
		} else {
			ignoreNames[dir] = struct{}{}
		}
	}
	return ignorePaths, ignoreNames, warnings
}

// scan returns the subdirectories of one scan directory, and warnings about any problems.
func scan(cfg *models.Config, scanDir models.ScanDir, flagDepth int, ignorePaths, ignoreNames models.StringSet) ([]string, []string) {
	root, err := utility.ResolvePath(scanDir.Path)
	if err != nil {
		return nil, []string{fmt.Sprintf("can't use scan_dirs path %s: %v", scanDir.Path, err)}
	}
	if utility.IsWithin(root, ignorePaths) {
		return nil, nil
	}
	if err := utility.RequireDirectory(root); err != nil {
		return nil, []string{fmt.Sprintf("can't scan: %v", err)}
	}

	depth := scanDir.GetDepth(flagDepth, cfg.Settings.DefaultDepth)
	subDirs, problems := utility.GetSubDirs(depth, root, ignoreNames, ignorePaths)

	var warnings []string
	for _, problem := range problems {
		warnings = append(warnings, fmt.Sprintf("can't scan everything: %v", problem))
	}
	return subDirs, warnings
}
