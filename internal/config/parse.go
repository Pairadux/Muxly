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
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	err := decoder.Decode(&cfg)
	if errors.Is(err, io.EOF) {
		// Nothing but comments, if that. Validate explains what's missing.
		return cfg, nil, nil
	}
	if err != nil {
		return cfg, removedKeyWarnings(&cfg), newParseError(err)
	}

	var extra yaml.Node
	err = decoder.Decode(&extra)
	switch {
	case errors.Is(err, io.EOF):
		return cfg, removedKeyWarnings(&cfg), nil
	case err != nil:
		return cfg, removedKeyWarnings(&cfg), newParseError(err)
	}
	return cfg, removedKeyWarnings(&cfg), &ParseError{
		Problems: []string{fmt.Sprintf("line %d: a second YAML document, which muxly would ignore", extra.Line)},
		Partial:  true,
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
