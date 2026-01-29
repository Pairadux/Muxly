package selector

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SanitizeSessionName creates a valid tmux session name from a directory name.
// Strips all leading dots, replaces middle dots with underscores, replaces colons with dashes.
// Returns the sanitized name and the count of leading dots stripped.
func SanitizeSessionName(name string) (string, int) {
	dotCount := 0
	for dotCount < len(name) && name[dotCount] == '.' {
		dotCount++
	}
	sanitized := name[dotCount:]
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "-")
	return sanitized, dotCount
}

// DotdirSuffix returns the appropriate suffix for a dotfile.
// Single dot: " [dotdir]", multiple dots: " [dotdir x2]", etc.
func DotdirSuffix(dotCount int) string {
	if dotCount == 0 {
		return ""
	}
	if dotCount == 1 {
		return " [dotdir]"
	}
	return fmt.Sprintf(" [dotdir x%d]", dotCount)
}

// SanitizePathForDisplay sanitizes the last component of a path by stripping
// leading dots and replacing middle dots/colons. This ensures display names
// match the actual session names that tmux creates.
func SanitizePathForDisplay(path string) string {
	if path == "" {
		return path
	}

	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 0 {
		lastIdx := len(parts) - 1
		sanitized, _ := SanitizeSessionName(parts[lastIdx])
		parts[lastIdx] = sanitized
	}
	return strings.Join(parts, string(filepath.Separator))
}

// ApplyPrefix adds an alias prefix to a display name if one is configured.
func ApplyPrefix(prefix, name string) string {
	if prefix != "" {
		return prefix + "/" + name
	}
	return name
}
