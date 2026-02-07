package selector

import (
	"fmt"
	"os"

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

	ignoreSet := b.buildIgnoreSet()
	allPaths := b.collectAllPaths(flagDepth, ignoreSet, currentSession)

	entries := make(map[string]models.DirEntry, len(allPaths)+len(existingSessions))
	b.addDirectoryEntries(entries, allPaths, currentSession, existingSessions)
	b.addTmuxSessionEntries(entries, existingSessions, currentSession)

	return entries, nil
}

// buildIgnoreSet creates a set of resolved paths from cfg.IgnoreDirs for O(1) lookup.
func (b *Builder) buildIgnoreSet() models.StringSet {
	ignoreSet := make(models.StringSet, len(b.cfg.IgnoreDirs))
	for _, dir := range b.cfg.IgnoreDirs {
		resolved, err := utility.ResolvePath(dir)
		if err == nil {
			ignoreSet[resolved] = struct{}{}
		}
	}
	return ignoreSet
}

// collectAllPaths gathers all directory paths from scan_dirs and entry_dirs.
func (b *Builder) collectAllPaths(flagDepth int, ignoreSet models.StringSet, currentSession string) []models.DirEntry {
	var allPaths []models.DirEntry

	addPath := func(path, prefix, template string) error {
		if _, ignored := ignoreSet[path]; ignored {
			return nil
		}

		entry := models.DirEntry{Path: path, Prefix: prefix, Template: template}
		allPaths = append(allPaths, entry)
		return nil
	}

	for _, scanDir := range b.cfg.ScanDirs {
		prefix := scanDir.Alias
		if err := b.processScanDir(scanDir, flagDepth, prefix, addPath); err != nil {
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
func (b *Builder) processScanDir(scanDir models.ScanDir, flagDepth int, prefix string, addEntry func(string, string, string) error) error {
	defaultDepth := b.cfg.Settings.DefaultDepth
	effectiveDepth := scanDir.GetDepth(flagDepth, defaultDepth)

	resolved, err := utility.ResolvePath(scanDir.Path)
	if err != nil {
		if b.verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to resolve scan directory %s: %v\n", scanDir.Path, err)
		}
		return nil
	}

	subDirs, err := utility.GetSubDirs(effectiveDepth, resolved)
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
