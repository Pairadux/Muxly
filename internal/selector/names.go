package selector

import (
	"path/filepath"
	"strings"
)

// SanitizeSessionName converts a directory name into a valid tmux session name.
//
// Tmux session names cannot start with dots, so this function strips leading dots.
func SanitizeSessionName(name string) string {
	return strings.TrimPrefix(name, ".")
}

// SanitizePathForDisplay sanitizes the last component of a path by stripping
// leading dots. This ensures display names match the actual session names that
// tmux creates.
func SanitizePathForDisplay(path string) string {
	if path == "" {
		return path
	}

	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 0 {
		lastIdx := len(parts) - 1
		parts[lastIdx] = SanitizeSessionName(parts[lastIdx])
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
