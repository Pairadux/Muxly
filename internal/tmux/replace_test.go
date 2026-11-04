//go:build unix

package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

// fakeTmux puts a logging tmux first on PATH, alone, so these tests can't reach a real
// server. Calls whose subcommand matches failOn exit 1. It returns a func reading the log.
func fakeTmux(t *testing.T, failOn string) func() []string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "log")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
[ "$1" = %q ] && exit 1
exit 0
`, log, failOn)
	if err := os.WriteFile(filepath.Join(dir, "tmux"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("TMUX", "/fake/socket,1,0")

	return func() []string {
		data, err := os.ReadFile(log)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			t.Fatal(err)
		}
		return strings.Split(strings.TrimSpace(string(data)), "\n")
	}
}

func replaceConfig() *models.Config {
	return &models.Config{
		Templates: []models.SessionTemplate{{
			Name:    "default",
			Default: true,
			Path:    "/work",
			Windows: []models.Window{{Name: "main"}, {Name: "term"}},
		}},
	}
}

func TestReplaceLastSession(t *testing.T) {
	parked := fmt.Sprintf("muxly-replaced-%d", os.Getpid())
	create := []string{
		"new-session -ds default -n main -c /work",
		"new-window -t default -n term -c /work",
	}
	enter := []string{
		"has-session -t default",
		"switch-client -t default:0",
	}

	tests := []struct {
		name    string
		current string
		failOn  string
		want    []string
		wantErr bool
	}{
		{
			name:    "different name: create, enter, then kill the old session",
			current: "other",
			want:    slices.Concat(create, enter, []string{"kill-session -t other"}),
		},
		{
			// Asking tmux whether "default" exists would match default-x by prefix,
			// switching to itself and then killing the last session.
			name:    "a name the template's is a prefix of is still replaced",
			current: "default-x",
			want:    slices.Concat(create, enter, []string{"kill-session -t default-x"}),
		},
		{
			name:    "same name: the old session gives up the name first",
			current: "default",
			want: slices.Concat(
				[]string{"rename-session -t =default " + parked},
				create, enter,
				[]string{"kill-session -t " + parked},
			),
		},
		{
			name:    "failing to rename changes nothing",
			current: "default",
			failOn:  "rename-session",
			want:    []string{"rename-session -t =default " + parked},
			wantErr: true,
		},
		{
			name:    "failing to create restores the old session and keeps it",
			current: "default",
			failOn:  "new-session",
			want: []string{
				"rename-session -t =default " + parked,
				"new-session -ds default -n main -c /work",
				"kill-session -t =default",
				"rename-session -t =" + parked + " default",
			},
			wantErr: true,
		},
		{
			name:    "failing to switch removes the new session and keeps the old one",
			current: "other",
			failOn:  "switch-client",
			want: slices.Concat(create, []string{
				"has-session -t default",
				"switch-client -t default:0",
				"switch-client -t default",
				"kill-session -t =default",
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := fakeTmux(t, tt.failOn)

			err := ReplaceLastSession(replaceConfig(), tt.current)

			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got := calls(); !slices.Equal(got, tt.want) {
				t.Errorf("tmux calls:\n got %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestReplaceLastSessionNoDefaultTemplate(t *testing.T) {
	calls := fakeTmux(t, "")

	if err := ReplaceLastSession(&models.Config{}, "current"); err == nil {
		t.Error("want an error")
	}
	if got := calls(); got != nil {
		t.Errorf("tmux was called: %q", got)
	}
}
