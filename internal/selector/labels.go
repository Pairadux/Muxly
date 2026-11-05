package selector

import (
	"path/filepath"
	"strings"
)

// assignLabels gives each directory the shortest label that no other directory has. A
// label is the directory's real name, after its alias if it has one. Directories with the
// same label each add a parent folder until they differ.
func assignLabels(dirs []Directory) {
	depths := make([]int, len(dirs))
	for i := range depths {
		depths[i] = 1
	}

	for grew := true; grew; {
		grew = false
		for _, group := range groupByLabel(dirs, depths) {
			if len(group) == 1 {
				continue
			}
			for _, i := range group {
				if depths[i] < len(pathComponents(dirs[i].Path)) {
					depths[i]++
					grew = true
				}
			}
		}
	}

	// Out of parents, an alias can still match a real folder: alias "proj" on /b reads the
	// same as /proj/b. Paths are unique, so they settle it.
	for _, group := range groupByLabel(dirs, depths) {
		for _, i := range group {
			if len(group) == 1 {
				dirs[i].Label = labelAtDepth(dirs[i], depths[i])
			} else {
				dirs[i].Label = dirs[i].Path
			}
		}
	}
}

// groupByLabel returns the indexes of dirs, grouped by the label each has at its depth.
func groupByLabel(dirs []Directory, depths []int) map[string][]int {
	groups := make(map[string][]int, len(dirs))
	for i, dir := range dirs {
		label := labelAtDepth(dir, depths[i])
		groups[label] = append(groups[label], i)
	}
	return groups
}

// labelAtDepth returns a directory's label using the last depth components of its path.
func labelAtDepth(dir Directory, depth int) string {
	components := pathComponents(dir.Path)
	if len(components) == 0 {
		return dir.Path
	}
	label := strings.Join(components[max(0, len(components)-depth):], string(filepath.Separator))
	if dir.Alias != "" {
		return dir.Alias + string(filepath.Separator) + label
	}
	return label
}

// pathComponents splits a path into its folders, so the root has none.
func pathComponents(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool { return r == filepath.Separator })
}
