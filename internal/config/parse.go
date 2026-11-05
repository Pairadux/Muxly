package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/Pairadux/muxly/internal/models"
	"go.yaml.in/yaml/v3"
)

// unknownKeyMessage matches how the yaml decoder reports a key that fits no field. A
// different wording only means the original message is shown, which still names the line.
var unknownKeyMessage = regexp.MustCompile(`^line (\d+): field (.+) not found in type \S+$`)

// ParseError lists what's wrong with the text of a config file.
type ParseError struct {
	// Problems holds one message per problem, each naming its line.
	Problems []string
	// Partial is true if the rest of the file could still be read, so the config returned
	// with this error is worth looking at.
	Partial bool
}

// Error returns the problems, one per line.
func (e *ParseError) Error() string {
	return strings.Join(e.Problems, "\n")
}

// Parse decodes the text of a config file. Anything it doesn't recognise is an error, so
// a misspelled key can't silently do nothing. It also returns warnings about what still
// loads but should be changed. After an error, the config holds whatever could be read.
func Parse(data []byte) (models.Config, []string, error) {
	var cfg models.Config
	err := decodeStrict(data, &cfg)
	spellOutHomePaths(&cfg)
	return cfg, removedKeyWarnings(&cfg), err
}

func decodeStrict(data []byte, cfg *models.Config) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var problems []string
	err := decoder.Decode(cfg)
	if errors.Is(err, io.EOF) {
		// Nothing but comments, if that. Validate explains what's missing.
		return nil
	}
	if err != nil {
		parseErr := newParseError(err)
		if !parseErr.Partial {
			return parseErr
		}
		problems = parseErr.Problems
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return newParseError(err)
	}
	problems = append(problems, emptyListItems(&doc)...)
	problems = append(problems, unknownEntryDirKeys(&doc)...)

	var extra yaml.Node
	err = decoder.Decode(&extra)
	switch {
	case errors.Is(err, io.EOF):
	case err != nil:
		return newParseError(err)
	default:
		problems = append(problems, fmt.Sprintf("line %d: a second YAML document, which muxly would ignore", extra.Line))
	}

	if len(problems) > 0 {
		return &ParseError{Problems: problems, Partial: true}
	}
	return nil
}

// emptyListItems finds list items with nothing in them, which the decoder drops without a
// word. "- ~" is the likely one, because an unquoted "~" means null in YAML, and it would
// lose someone's home directory from the picker.
func emptyListItems(node *yaml.Node) []string {
	var problems []string
	for _, child := range node.Content {
		if node.Kind == yaml.SequenceNode && child.ShortTag() == "!!null" {
			problems = append(problems, fmt.Sprintf(`line %d: an empty list item, which would be ignored. An unquoted ~ counts as empty, so write "~" for your home directory`, child.Line))
		}
		problems = append(problems, emptyListItems(child)...)
	}
	return problems
}

// unknownEntryDirKeys does for entry_dirs what strict decoding does everywhere else. It
// can't do it there, because EntryDir decodes itself to allow for the short form. Reporting
// these here rather than from EntryDir also keeps the item, where an error would drop it,
// and doctor would go on to say there were no directories.
func unknownEntryDirKeys(doc *yaml.Node) []string {
	if len(doc.Content) == 0 {
		return nil
	}
	_, list := findKey(doc.Content[0], "entry_dirs")
	if list == nil {
		return nil
	}
	var problems []string
	for _, item := range list.Content {
		if item.Kind != yaml.MappingNode {
			continue
		}
		for i := 0; i+1 < len(item.Content); i += 2 {
			switch key := item.Content[i]; key.Value {
			case "path", "template":
			default:
				problems = append(problems, fmt.Sprintf("line %d: unknown key %q", key.Line, key.Value))
			}
		}
	}
	return problems
}

// spellOutHomePaths undoes a YAML surprise: an unquoted "~" means null, so "path: ~" decodes
// to an empty path. That still opens the home directory, but anything showing the path
// would show nothing.
func spellOutHomePaths(cfg *models.Config) {
	for i := range cfg.ScanDirs {
		if cfg.ScanDirs[i].Path == "" {
			cfg.ScanDirs[i].Path = "~"
		}
	}
	for i := range cfg.EntryDirs {
		if cfg.EntryDirs[i].Path == "" {
			cfg.EntryDirs[i].Path = "~"
		}
	}
}

// newParseError rewords a decoding error. The decoder carries on past a value it can't
// use and reports them all together, but a syntax error stops it with nothing decoded.
func newParseError(err error) *ParseError {
	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		return &ParseError{Problems: []string{"invalid YAML: " + strings.TrimPrefix(err.Error(), "yaml: ")}}
	}
	problems := make([]string, len(typeErr.Errors))
	for i, problem := range typeErr.Errors {
		problems[i] = unknownKeyMessage.ReplaceAllString(problem, `line $1: unknown key "$2"`)
	}
	return &ParseError{Problems: problems, Partial: true}
}

// removedKeyWarnings reports keys that used to mean something and are now ignored.
func removedKeyWarnings(cfg *models.Config) []string {
	if cfg.Settings.RemovedTmuxBase.IsZero() {
		return nil
	}
	return []string{fmt.Sprintf("settings.tmux_base on line %d was removed in 0.27.0 and does nothing, so delete that line", cfg.Settings.RemovedTmuxBase.Line)}
}
