package selector

import (
	"maps"
	"path/filepath"
	"strings"

	"github.com/Pairadux/muxly/internal/models"
)

// DeduplicateDisplayNames creates unique display names for paths with conflicting basenames.
//
// When multiple paths have the same basename (e.g., ~/Dev/project1/src and ~/Work/project2/src),
// this function finds the minimum path suffix needed to make them distinguishable.
func DeduplicateDisplayNames(allPaths []models.PathInfo) map[string]string {
	if len(allPaths) == 0 {
		return make(map[string]string)
	}

	groups := make(map[string][]models.PathInfo, len(allPaths))
	for _, info := range allPaths {
		basename := filepath.Base(info.Path)
		groups[basename] = append(groups[basename], info)
	}

	result := make(map[string]string, len(allPaths))

	for _, group := range groups {
		if len(group) == 1 {
			info := group[0]
			displayName := NormalizeSessionName(filepath.Base(info.Path))
			displayName = ApplyPrefix(info.Prefix, displayName)
			result[info.Path] = displayName
		} else {
			resolved := resolveConflicts(group)
			maps.Copy(result, resolved)
		}
	}

	return result
}

// resolveConflicts finds the minimum suffix depth needed to make all paths unique.
func resolveConflicts(paths []models.PathInfo) map[string]string {
	const maxDepth = 10

	for depth := 1; depth <= maxDepth; depth++ {
		suffixes := make(map[string]models.PathInfo, len(paths))
		conflicts := false

		for _, info := range paths {
			suffix := getPathSuffix(info.Path, depth)
			if existing, exists := suffixes[suffix]; exists {
				if existing.Path != info.Path {
					conflicts = true
					break
				}
			}
			suffixes[suffix] = info
		}

		if !conflicts {
			result := make(map[string]string, len(suffixes))
			for suffix, info := range suffixes {
				displayName := NormalizePathForDisplay(suffix)
				displayName = ApplyPrefix(info.Prefix, displayName)
				result[info.Path] = displayName
			}
			return result
		}
	}

	result := make(map[string]string, len(paths))
	for _, info := range paths {
		displayName := NormalizePathForDisplay(info.Path)
		displayName = ApplyPrefix(info.Prefix, displayName)
		result[info.Path] = displayName
	}
	return result
}

// getPathSuffix extracts the last N components of a path for display purposes.
func getPathSuffix(path string, depth int) string {
	components := strings.Split(filepath.Clean(path), string(filepath.Separator))

	cleanComponents := make([]string, 0, len(components))
	for _, comp := range components {
		if comp != "" {
			cleanComponents = append(cleanComponents, comp)
		}
	}

	if depth >= len(cleanComponents) {
		return strings.Join(cleanComponents, string(filepath.Separator))
	}

	start := len(cleanComponents) - depth
	return strings.Join(cleanComponents[start:], string(filepath.Separator))
}
