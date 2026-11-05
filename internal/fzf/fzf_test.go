//go:build unix

package fzf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// fakeFzf puts a script named fzf first on PATH that saves what it's given, prints output
// and exits with code. It returns the path of the saved input.
func fakeFzf(t *testing.T, output string, code int) string {
	t.Helper()
	dir := t.TempDir()
	input := filepath.Join(dir, "input")
	script := fmt.Sprintf("#!/bin/sh\n/bin/cat > %q\nprintf '%%b\\n' %q\nexit %d\n", input, output, code)
	if err := os.WriteFile(filepath.Join(dir, "fzf"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return input
}

func TestPick(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		code          int
		want          int
		wantCancelled bool
		wantErr       bool
	}{
		{name: "the index of the chosen row", output: `1\tb`, want: 1},
		{name: "the first row is not a cancel", output: `0\ta`, want: 0},
		{name: "ctrl-c or esc is a cancel", code: 130, wantCancelled: true},
		{name: "enter with no match is a cancel", code: 1, wantCancelled: true},
		{name: "fzf failing is a real error", code: 2, wantErr: true},
		{name: "an index that isn't there is an error", output: `5\tz`, wantErr: true},
		{name: "a negative index is an error", output: `-1\tz`, wantErr: true},
		{name: "a line with no index is an error", output: "z", wantErr: true},
		{name: "no output is an error", output: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeFzf(t, tt.output, tt.code)

			got, err := Pick([]string{"a", "b"})

			if cancelled := errors.Is(err, ErrCancelled); cancelled != tt.wantCancelled {
				t.Fatalf("errors.Is(err, ErrCancelled) = %v, want %v (err: %v)", cancelled, tt.wantCancelled, err)
			}
			if tt.wantErr && (err == nil || errors.Is(err, ErrCancelled)) {
				t.Fatalf("want a real error, got %v", err)
			}
			if !tt.wantErr && !tt.wantCancelled && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %d, want %d", got, tt.want)
			}
		})
	}
}

// A label holding a tab or a newline would otherwise split a row or shift its fields.
func TestPickKeepsEachLabelOnOneLine(t *testing.T) {
	input := fakeFzf(t, `0\tz`, 0)

	if _, err := Pick([]string{"plain", "tab\there", "new\nline", "escape\x1b[31m"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(input)
	if err != nil {
		t.Fatal(err)
	}
	want := "0\tplain\n1\ttab?here\n2\tnew?line\n3\tescape?[31m\n"
	if string(data) != want {
		t.Errorf("fzf was given %q, want %q", data, want)
	}
}

func TestSelectWithFzf(t *testing.T) {
	fakeFzf(t, `1\tb`, 0)
	if got, err := SelectWithFzf([]string{"a", "b"}); err != nil || got != "b" {
		t.Errorf("got %q, %v, want b", got, err)
	}

	fakeFzf(t, "", 130)
	if _, err := SelectWithFzf([]string{"a", "b"}); !errors.Is(err, ErrCancelled) {
		t.Errorf("got %v, want ErrCancelled", err)
	}
}
