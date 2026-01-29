package selector

import (
	"fmt"
	"maps"
	"path/filepath"
	"strings"

	"github.com/Pairadux/muxly/internal/models"
)

type dedupeEntry struct {
	info     models.PathInfo
	dotCount int
}

// DeduplicateDisplayNames creates unique display names for paths with conflicting basenames.
//
// When multiple paths have the same basename (e.g., ~/Dev/project1/src and ~/Work/project2/src),
// this function finds the minimum path suffix needed to make them distinguishable.
// Aliases are only applied when deduplication is needed.
// Dotfiles always get a [dotdir] suffix (with count for multiple leading dots).
func DeduplicateDisplayNames(allPaths []models.PathInfo) map[string]string {
	if len(allPaths) == 0 {
		return make(map[string]string)
	}

	entries := buildDedupeEntries(allPaths)
	groupedByBasename := groupByBasename(entries)

	result := make(map[string]string, len(allPaths))
	for _, group := range groupedByBasename {
		if len(group) == 1 {
			entry := group[0]
			sanitizedName, _ := SanitizeSessionName(filepath.Base(entry.info.Path))
			displayName := sanitizedName + DotdirSuffix(entry.dotCount)
			result[entry.info.Path] = displayName
		} else {
			maps.Copy(result, disambiguate(group))
		}
	}
	return result
}

func buildDedupeEntries(allPaths []models.PathInfo) []dedupeEntry {
	entries := make([]dedupeEntry, len(allPaths))
	for i, info := range allPaths {
		basename := filepath.Base(info.Path)
		_, dotCount := SanitizeSessionName(basename)
		entries[i] = dedupeEntry{
			info:     info,
			dotCount: dotCount,
		}
	}
	return entries
}

func groupByBasename(entries []dedupeEntry) map[string][]dedupeEntry {
	groups := make(map[string][]dedupeEntry)
	for _, e := range entries {
		sanitizedName, _ := SanitizeSessionName(filepath.Base(e.info.Path))
		groups[sanitizedName] = append(groups[sanitizedName], e)
	}
	return groups
}

func disambiguate(entries []dedupeEntry) map[string]string {
	const maxDepth = 10

	for depth := 1; depth <= maxDepth; depth++ {
		nameGroups := groupEntriesByDisplayName(entries, depth)

		if resolved, ok := finalizeIfUnique(nameGroups); ok {
			return resolved
		}
	}

	return useFullPaths(entries)
}

func groupEntriesByDisplayName(entries []dedupeEntry, depth int) map[string][]dedupeEntry {
	groups := make(map[string][]dedupeEntry)
	for _, e := range entries {
		name := displayNameAtDepth(e, depth)
		groups[name] = append(groups[name], e)
	}
	return groups
}

func displayNameAtDepth(e dedupeEntry, depth int) string {
	sanitizedName, _ := SanitizeSessionName(filepath.Base(e.info.Path))

	if depth == 1 {
		if e.info.Prefix != "" {
			return e.info.Prefix + "/" + sanitizedName
		}
		return sanitizedName
	}

	suffix := SanitizePathForDisplay(getPathSuffix(e.info.Path, depth-1))
	if e.info.Prefix != "" {
		return e.info.Prefix + "/" + suffix
	}

	return SanitizePathForDisplay(getPathSuffix(e.info.Path, depth))
}

func finalizeIfUnique(nameGroups map[string][]dedupeEntry) (map[string]string, bool) {
	result := make(map[string]string)

	for displayName, group := range nameGroups {
		if len(group) == 1 {
			entry := group[0]
			result[entry.info.Path] = displayName + DotdirSuffix(entry.dotCount)
			continue
		}

		allSameDotCount := true
		firstDotCount := group[0].dotCount
		for _, e := range group[1:] {
			if e.dotCount != firstDotCount {
				allSameDotCount = false
				break
			}
		}

		if !allSameDotCount {
			for _, e := range group {
				result[e.info.Path] = displayName + DotdirSuffix(e.dotCount)
			}
			continue
		}

		return nil, false
	}
	return result, true
}

func useFullPaths(entries []dedupeEntry) map[string]string {
	result := make(map[string]string, len(entries))
	seen := make(map[string]int)

	for _, e := range entries {
		displayName := SanitizePathForDisplay(e.info.Path) + DotdirSuffix(e.dotCount)

		if count := seen[displayName]; count > 0 {
			displayName = fmt.Sprintf("%s (%d)", displayName, count+1)
		}
		seen[displayName]++

		result[e.info.Path] = displayName
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
