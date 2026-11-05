package selector

import (
	"maps"
	"testing"
)

func TestAssignLabels(t *testing.T) {
	tests := []struct {
		name string
		dirs []Directory
		want map[string]string
	}{
		{name: "no directories", want: map[string]string{}},
		{
			name: "the directory's real spelling",
			dirs: []Directory{{Path: "/home/me/Dev/MyProject"}, {Path: "/home/me/.config"}, {Path: "/home/me/my.project"}},
			want: map[string]string{
				"/home/me/Dev/MyProject": "MyProject",
				"/home/me/.config":       ".config",
				"/home/me/my.project":    "my.project",
			},
		},
		{
			name: "same name adds the parent",
			dirs: []Directory{{Path: "/Dev/muxly"}, {Path: "/Work/muxly"}, {Path: "/Projects/muxly"}},
			want: map[string]string{
				"/Dev/muxly":      "Dev/muxly",
				"/Work/muxly":     "Work/muxly",
				"/Projects/muxly": "Projects/muxly",
			},
		},
		{
			name: "same parent adds the next one",
			dirs: []Directory{{Path: "/a/shared/src"}, {Path: "/b/shared/src"}},
			want: map[string]string{
				"/a/shared/src": "a/shared/src",
				"/b/shared/src": "b/shared/src",
			},
		},
		{
			name: "only the directories that clash get longer",
			dirs: []Directory{{Path: "/a/shared/src"}, {Path: "/b/shared/src"}, {Path: "/c/other/src"}, {Path: "/d/lib"}},
			want: map[string]string{
				"/a/shared/src": "a/shared/src",
				"/b/shared/src": "b/shared/src",
				"/c/other/src":  "other/src",
				"/d/lib":        "lib",
			},
		},
		{
			name: "dots no longer make names collide",
			dirs: []Directory{{Path: "/a/.src"}, {Path: "/b/.src"}, {Path: "/c/..src"}, {Path: "/d/src"}},
			want: map[string]string{
				"/a/.src":  "a/.src",
				"/b/.src":  "b/.src",
				"/c/..src": "..src",
				"/d/src":   "src",
			},
		},
		{
			name: "names that differ only by case stay apart",
			dirs: []Directory{{Path: "/x/Foo"}, {Path: "/x/foo"}},
			want: map[string]string{"/x/Foo": "Foo", "/x/foo": "foo"},
		},
		{
			name: "an alias always shows",
			dirs: []Directory{{Path: "/projects/unique", Alias: "proj"}},
			want: map[string]string{"/projects/unique": "proj/unique"},
		},
		{
			name: "an alias separates two directories without adding a parent",
			dirs: []Directory{{Path: "/config/muxly", Alias: "cfg"}, {Path: "/Dev/muxly"}},
			want: map[string]string{"/config/muxly": "cfg/muxly", "/Dev/muxly": "muxly"},
		},
		{
			name: "the same alias adds parents after it",
			dirs: []Directory{{Path: "/p/a/src", Alias: "proj"}, {Path: "/p/b/src", Alias: "proj"}},
			want: map[string]string{"/p/a/src": "proj/a/src", "/p/b/src": "proj/b/src"},
		},
		{
			name: "a directory at the top keeps its name beside a longer one",
			dirs: []Directory{{Path: "/src"}, {Path: "/a/src"}},
			want: map[string]string{"/src": "src", "/a/src": "a/src"},
		},
		{
			name: "an alias that reads like a real folder falls back to paths",
			dirs: []Directory{{Path: "/b", Alias: "proj"}, {Path: "/proj/b"}, {Path: "/z/b"}},
			want: map[string]string{"/b": "/b", "/proj/b": "/proj/b", "/z/b": "z/b"},
		},
		{
			name: "the filesystem root",
			dirs: []Directory{{Path: "/"}, {Path: "/usr"}},
			want: map[string]string{"/": "/", "/usr": "usr"},
		},
		{
			name: "spaces",
			dirs: []Directory{{Path: "/My Projects/muxly"}, {Path: "/Dev/muxly"}},
			want: map[string]string{"/My Projects/muxly": "My Projects/muxly", "/Dev/muxly": "Dev/muxly"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assignLabels(tt.dirs)

			got := make(map[string]string, len(tt.dirs))
			for _, dir := range tt.dirs {
				got[dir.Path] = dir.Label
			}
			if !maps.Equal(got, tt.want) {
				t.Errorf("labels = %q, want %q", got, tt.want)
			}
		})
	}
}
