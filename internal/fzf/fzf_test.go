package fzf

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// fakeFzf puts a script named fzf first on PATH that prints output and exits with code.
func fakeFzf(t *testing.T, output string, code int) {
	t.Helper()
	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\ncat >/dev/null\nprintf '%%s\\n' %q\nexit %d\n", output, code)
	if err := os.WriteFile(filepath.Join(dir, "fzf"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
}

func TestSelectWithFzf(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		code          int
		want          string
		wantCancelled bool
		wantErr       bool
	}{
		{name: "selection is returned trimmed", output: "chosen", code: 0, want: "chosen"},
		{name: "ctrl-c or esc is a cancel", code: 130, wantCancelled: true},
		{name: "fzf failing is a real error", code: 2, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeFzf(t, tt.output, tt.code)

			got, err := SelectWithFzf([]string{"a", "b"})

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
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
