package selector

import (
	"slices"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

func TestSortedNames(t *testing.T) {
	const prefix = "[TMUX] "

	tests := []struct {
		name    string
		entries []string
		want    []string
	}{
		{name: "no entries", want: []string{}},
		{
			name:    "sessions come before directories",
			entries: []string{"apple", prefix + "zebra", "mango", prefix + "kiwi"},
			want:    []string{prefix + "kiwi", prefix + "zebra", "apple", "mango"},
		},
		{
			name:    "order ignores case",
			entries: []string{"Beta", "alpha", "Gamma"},
			want:    []string{"alpha", "Beta", "Gamma"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := make(map[string]models.DirEntry, len(tt.entries))
			for _, name := range tt.entries {
				entries[name] = models.DirEntry{}
			}

			if got := SortedNames(entries, prefix); !slices.Equal(got, tt.want) {
				t.Errorf("SortedNames() = %q, want %q", got, tt.want)
			}
		})
	}
}
