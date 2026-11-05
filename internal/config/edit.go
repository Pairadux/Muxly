package config

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/utility"
	"go.yaml.in/yaml/v3"
)

// emptyListLine matches a top-level key whose list has nothing in it yet, keeping the key
// and any comment: "scan_dirs: [] # later".
var emptyListLine = regexp.MustCompile(`^([^\s:#]+:)\s*(?:\[\s*\]|~|null|Null|NULL)?\s*(#.*)?\s*$`)

// listEdit changes the text of one top-level list, given the file split into lines and its
// parsed form, which knows what's on which line.
type listEdit func(lines []string, root *yaml.Node, key string) (string, error)

// AddEntryDir adds dir to entry_dirs in the config file at path.
func AddEntryDir(path string, dir models.EntryDir) error {
	return editList(path, "entry_dirs", appendItem(dir), func(cfg *models.Config) {
		cfg.EntryDirs = append(cfg.EntryDirs, dir)
	})
}

// AddScanDir adds dir to scan_dirs in the config file at path.
func AddScanDir(path string, dir models.ScanDir) error {
	return editList(path, "scan_dirs", appendItem(dir), func(cfg *models.Config) {
		cfg.ScanDirs = append(cfg.ScanDirs, dir)
	})
}

// RemoveEntryDir removes item number index of entry_dirs from the config file at path.
func RemoveEntryDir(path string, index int) error {
	return editList(path, "entry_dirs", removeItem(index), func(cfg *models.Config) {
		cfg.EntryDirs = slices.Delete(cfg.EntryDirs, index, index+1)
	})
}

// RemoveScanDir removes item number index of scan_dirs from the config file at path.
func RemoveScanDir(path string, index int) error {
	return editList(path, "scan_dirs", removeItem(index), func(cfg *models.Config) {
		cfg.ScanDirs = slices.Delete(cfg.ScanDirs, index, index+1)
	})
}

// editList changes one list in the config file and leaves every other byte alone.
//
// Encoding a parsed file back to YAML would keep most comments, but it drops blank lines
// and moves some comments, so the parsed form is only used to find where things are, and
// the lines are changed in the original text. That can go wrong on YAML written in an
// unusual way, so the result has to decode to exactly the config that was asked for, and
// be valid, or nothing is written.
func editList(path, key string, edit listEdit, apply func(*models.Config)) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	want, _, err := Parse(data)
	if err != nil {
		return fmt.Errorf("invalid config file %s\n%w", path, err)
	}
	apply(&want)

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("invalid config file %s\n%w", path, err)
	}
	var root *yaml.Node
	if len(doc.Content) > 0 {
		root = doc.Content[0]
	}

	edited, err := edit(strings.SplitAfter(string(data), "\n"), root, key)
	if err != nil {
		return err
	}

	got, _, err := Parse([]byte(edited))
	if err != nil || !sameConfig(got, want) {
		return errNotEditable(key)
	}
	if err := Validate(&got); err != nil {
		return fmt.Errorf("that would stop the config working, so it was left as it is:\n%w", err)
	}

	return utility.WriteFileAtomic(path, []byte(edited), constants.FilePermissions)
}

func errNotEditable(key string) error {
	return fmt.Errorf("can't change %s the way it's written in the config, so it was left as it is\nMake the change with 'muxly config edit'", key)
}

// sameConfig compares two configs by what they'd be written as, so that an empty list and
// a missing one count as the same.
func sameConfig(a, b models.Config) bool {
	aText, aErr := yaml.Marshal(a)
	bText, bErr := yaml.Marshal(b)
	return aErr == nil && bErr == nil && bytes.Equal(aText, bText)
}

// appendItem adds value at the end of the list, starting one if there's no list yet.
func appendItem(value any) listEdit {
	return func(lines []string, root *yaml.Node, key string) (string, error) {
		keyNode, list := findKey(root, key)

		if keyNode == nil {
			item, err := renderItem(value, preferredIndent(lines, root))
			if err != nil {
				return "", err
			}
			return insertAfter(lines, len(lines), key+":\n"+item), nil
		}

		if isEmptyList(list) {
			if keyNode.Line > len(lines) {
				return "", errNotEditable(key)
			}
			match := emptyListLine.FindStringSubmatch(lines[keyNode.Line-1])
			if match == nil {
				return "", errNotEditable(key)
			}
			keyLine := match[1]
			if match[2] != "" {
				keyLine += " " + match[2]
			}
			item, err := renderItem(value, preferredIndent(lines, root))
			if err != nil {
				return "", err
			}
			updated := slices.Clone(lines)
			updated[keyNode.Line-1] = keyLine + "\n" + item
			return strings.Join(updated, ""), nil
		}

		if !isBlockList(list) {
			return "", errNotEditable(key)
		}
		last := list.Content[len(list.Content)-1]
		indent, ok := itemIndent(lines, last)
		if !ok || lastLine(last) > len(lines) {
			return "", errNotEditable(key)
		}
		item, err := renderItem(value, indent)
		if err != nil {
			return "", err
		}
		return insertAfter(lines, lastLine(last), item), nil
	}
}

// removeItem deletes the lines of one list item. Comments on lines of their own stay,
// since there's no telling which item they were about.
func removeItem(index int) listEdit {
	return func(lines []string, root *yaml.Node, key string) (string, error) {
		_, list := findKey(root, key)
		if list == nil || !isBlockList(list) || index < 0 || index >= len(list.Content) {
			return "", errNotEditable(key)
		}
		item := list.Content[index]
		if _, ok := itemIndent(lines, item); !ok || lastLine(item) > len(lines) {
			return "", errNotEditable(key)
		}
		return strings.Join(slices.Delete(slices.Clone(lines), item.Line-1, lastLine(item)), ""), nil
	}
}

// findKey returns a top-level key and its value, or nil for both if it isn't there.
func findKey(root *yaml.Node, key string) (keyNode, value *yaml.Node) {
	if root == nil {
		return nil, nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i], root.Content[i+1]
		}
	}
	return nil, nil
}

func isEmptyList(node *yaml.Node) bool {
	return node.ShortTag() == "!!null" || (node.Kind == yaml.SequenceNode && len(node.Content) == 0)
}

// isBlockList reports whether node is a list with one "- item" per line, rather than
// written inline as "[a, b]".
func isBlockList(node *yaml.Node) bool {
	return node.Kind == yaml.SequenceNode && node.Style&yaml.FlowStyle == 0 && len(node.Content) > 0
}

// itemIndent returns the whitespace before a list item's dash. It reports false if the
// item's line doesn't start with one, which means the dash is on a line of its own.
func itemIndent(lines []string, item *yaml.Node) (string, bool) {
	if item.Line > len(lines) {
		return "", false
	}
	line := lines[item.Line-1]
	rest := strings.TrimLeft(line, " ")
	if !strings.HasPrefix(rest, "-") {
		return "", false
	}
	return line[:len(line)-len(rest)], true
}

// preferredIndent returns the indent to start a new list with: that of the first list in
// the file, so it matches how the file is written, or else two spaces.
func preferredIndent(lines []string, root *yaml.Node) string {
	if root != nil {
		for i := 1; i < len(root.Content); i += 2 {
			if list := root.Content[i]; isBlockList(list) {
				if indent, ok := itemIndent(lines, list.Content[0]); ok {
					return indent
				}
			}
		}
	}
	return "  "
}

// lastLine returns the last line a node starts anything on. A value that runs over several
// lines goes past it, and the check in editList catches what that breaks.
func lastLine(node *yaml.Node) int {
	line := node.Line
	for _, child := range node.Content {
		line = max(line, lastLine(child))
	}
	return line
}

// renderItem writes value as a list item at the given indent. Marshalling takes care of
// anything that needs quoting, such as "~", which on its own means null in YAML.
func renderItem(value any, indent string) (string, error) {
	text, err := yaml.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("writing the new item: %w", err)
	}
	var item strings.Builder
	for i, line := range strings.Split(strings.TrimRight(string(text), "\n"), "\n") {
		marker := "  "
		if i == 0 {
			marker = "- "
		}
		item.WriteString(indent + marker + line + "\n")
	}
	return item.String(), nil
}

// insertAfter returns the text with added placed after line number n, counting from 1.
func insertAfter(lines []string, n int, added string) string {
	before := strings.Join(lines[:n], "")
	if before != "" && !strings.HasSuffix(before, "\n") {
		before += "\n"
	}
	return before + added + strings.Join(lines[n:], "")
}
