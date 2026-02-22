package selector

import (
	"fmt"
	"os"
	"strings"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/tmux"
	"github.com/Pairadux/muxly/internal/utility"
)

// Builder constructs selector entries from configuration
type Builder struct {
	cfg     *models.Config
	verbose bool
}

// NewBuilder creates a new Builder with the given configuration
func NewBuilder(cfg *models.Config, verbose bool) *Builder {
	return &Builder{
		cfg:     cfg,
		verbose: verbose,
	}
}

// BuildEntries creates a map of display names to directory paths by
// processing scan_dirs and entry_dirs from the configuration. It handles
// directory scanning at specified depths, filters out ignored directories,
// excludes the current tmux session, and marks existing tmux sessions with
// a prefix.
//
// The flagDepth parameter can override the scanning depth for scan_dirs.
// Returns a map where keys are display names and values are resolved paths
// or session names for existing tmux sessions.
func (b *Builder) BuildEntries(flagDepth int) (map[string]models.DirEntry, error) {
	existingSessions := tmux.GetTmuxSessionSet()
	currentSession := tmux.GetCurrentTmuxSession()

	ignorePaths, ignoreNames := b.buildIgnoreSets()
	allPaths := b.collectAllPaths(flagDepth, ignorePaths, ignoreNames, currentSession)

	entries := make(map[string]models.DirEntry, len(allPaths)+len(existingSessions))
	b.addDirectoryEntries(entries, allPaths, currentSession, existingSessions)
	b.addTmuxSessionEntries(entries, existingSessions, currentSession)

	return entries, nil
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
//	  - ~/projects/old    # path       — matches only that specific resolved directory
func (b *Builder) buildIgnoreSets() (ignorePaths models.StringSet, ignoreNames models.StringSet) {
	ignorePaths = make(models.StringSet)
	ignoreNames = make(models.StringSet)

	for _, dir := range config.BaseIgnoreDirs {
		ignoreNames[dir] = struct{}{}
	}

	for _, dir := range b.cfg.IgnoreDirs {
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

// collectAllPaths gathers all directory paths from scan_dirs and entry_dirs.
// ignorePaths filters by resolved absolute path; ignoreNames is passed to GetSubDirs
// for basename-level filtering during the walk itself.
func (b *Builder) collectAllPaths(flagDepth int, ignorePaths, ignoreNames models.StringSet, currentSession string) []models.DirEntry {
	var allPaths []models.DirEntry

	addPath := func(path, prefix, template string) error {
		if _, ignored := ignorePaths[path]; ignored {
			return nil
		}

		entry := models.DirEntry{Path: path, Prefix: prefix, Template: template}
		allPaths = append(allPaths, entry)
		return nil
	}

	for _, scanDir := range b.cfg.ScanDirs {
		prefix := scanDir.Alias
		if err := b.processScanDir(scanDir, flagDepth, prefix, ignoreNames, addPath); err != nil {
			continue
		}
	}

	for _, entryDir := range b.cfg.EntryDirs {
		resolved, err := utility.ResolvePath(entryDir.Path)
		if err != nil {
			if b.verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to resolve entry directory %s: %v\n", entryDir.Path, err)
			}
			continue
		}
		addPath(resolved, "", entryDir.Template)
	}

	return allPaths
}

// addDirectoryEntries populates the entries map with display names for directories.
func (b *Builder) addDirectoryEntries(entries map[string]models.DirEntry, allPaths []models.DirEntry, currentSession string, existingSessions map[string]bool) {
	displayNames := DeduplicateDisplayNames(allPaths)

	for _, info := range allPaths {
		displayName := displayNames[info.Path]

		if shouldSkipEntry(displayName, currentSession, existingSessions) {
			continue
		}

		entries[displayName] = info
	}
}

// addTmuxSessionEntries adds existing tmux sessions to the entries map.
func (b *Builder) addTmuxSessionEntries(entries map[string]models.DirEntry, existingSessions map[string]bool, currentSession string) {
	for sessionName := range existingSessions {
		if sessionName == currentSession {
			continue
		}

		displayName := b.cfg.Settings.TmuxSessionPrefix + sessionName
		entries[displayName] = models.DirEntry{Path: sessionName}
	}
}

// processScanDir scans a single scan_dir entry and adds all discovered subdirectories.
func (b *Builder) processScanDir(scanDir models.ScanDir, flagDepth int, prefix string, ignoreNames models.StringSet, addEntry func(string, string, string) error) error {
	defaultDepth := b.cfg.Settings.DefaultDepth
	effectiveDepth := scanDir.GetDepth(flagDepth, defaultDepth)

	resolved, err := utility.ResolvePath(scanDir.Path)
	if err != nil {
		if b.verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to resolve scan directory %s: %v\n", scanDir.Path, err)
		}
		return nil
	}

	subDirs, err := utility.GetSubDirs(effectiveDepth, resolved, ignoreNames)
	if err != nil {
		if b.verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", resolved, err)
		}
		return nil
	}

	for _, subDir := range subDirs {
		if err := addEntry(subDir, prefix, scanDir.Template); err != nil {
			return err
		}
	}

	return nil
}

// shouldSkipEntry determines if a directory entry should be excluded from the selector.
func shouldSkipEntry(displayName, currentSession string, existingSessions map[string]bool) bool {
	return displayName == currentSession || existingSessions[displayName]
}
