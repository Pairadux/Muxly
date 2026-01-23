package selector

import (
	"path/filepath"
	"strings"
)

// NormalizeSessionName converts a directory name into a valid tmux session name.
//
// Tmux session names cannot start with dots, so this function replaces leading dots
// with underscores.
func NormalizeSessionName(name string) string {
	if strings.HasPrefix(name, ".") {
		return "_" + name[1:]
	}
	return name
}

// NormalizePathForDisplay normalizes the last component of a path by replacing
// leading dots with underscores. This ensures display names match the actual
// session names that tmux creates.
func NormalizePathForDisplay(path string) string {
	if path == "" {
		return path
	}

	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 0 {
		lastIdx := len(parts) - 1
		parts[lastIdx] = NormalizeSessionName(parts[lastIdx])
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
