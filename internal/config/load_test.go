package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesEachEnvironmentVariable(t *testing.T) {
	const file = "entry_dirs:\n  - path: ~/a\nsettings:\n  editor: file-editor\n  default_depth: 2\n  tmux_session_prefix: '[FILE] '\n"

	tests := []struct {
		name  string
		key   string
		value string
		check func(editor string, depth int, prefix string, alwaysKill bool) bool
	}{
		{
			name: "editor", key: "MUXLY_EDITOR", value: "code --wait",
			check: func(editor string, _ int, _ string, _ bool) bool { return editor == "code --wait" },
		},
		{
			name: "blank editor counts as unset", key: "MUXLY_EDITOR", value: "  ",
			check: func(editor string, _ int, _ string, _ bool) bool { return editor == "file-editor" },
		},
		{
			name: "depth", key: "MUXLY_DEFAULT_DEPTH", value: "7",
			check: func(_ string, depth int, _ string, _ bool) bool { return depth == 7 },
		},
		{
			name: "empty depth counts as unset", key: "MUXLY_DEFAULT_DEPTH", value: "",
			check: func(_ string, depth int, _ string, _ bool) bool { return depth == 2 },
		},
		{
			name: "prefix, spaces and all", key: "MUXLY_TMUX_SESSION_PREFIX", value: "[WORK] ",
			check: func(_ string, _ int, prefix string, _ bool) bool { return prefix == "[WORK] " },
		},
		{
			name: "empty prefix counts as unset", key: "MUXLY_TMUX_SESSION_PREFIX", value: "",
			check: func(_ string, _ int, prefix string, _ bool) bool { return prefix == "[FILE] " },
		},
		{
			name: "always kill", key: "MUXLY_ALWAYS_KILL_ON_LAST_SESSION", value: "true",
			check: func(_ string, _ int, _ string, alwaysKill bool) bool { return alwaysKill },
		},
		{
			name: "always kill off", key: "MUXLY_ALWAYS_KILL_ON_LAST_SESSION", value: "false",
			check: func(_ string, _ int, _ string, alwaysKill bool) bool { return !alwaysKill },
		},
		{
			name: "what viper happened to accept does nothing", key: "MUXLY_SETTINGS.DEFAULT_DEPTH", value: "7",
			check: func(_ string, depth int, _ string, _ bool) bool { return depth == 2 },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)

			cfg, report := Load(writeConfigFile(t, file))

			if err := report.Err(); err != nil {
				t.Fatal(err)
			}
			s := cfg.Settings
			if !tt.check(s.Editor, s.DefaultDepth, s.TmuxSessionPrefix, s.AlwaysKillOnLastSession) {
				t.Errorf("settings = %+v", s)
			}
		})
	}
}

func TestLoadRejectsEnvironmentVariablesItCannotUse(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{key: "MUXLY_DEFAULT_DEPTH", value: "abc"},
		{key: "MUXLY_DEFAULT_DEPTH", value: "0"},
		{key: "MUXLY_DEFAULT_DEPTH", value: "-1"},
		{key: "MUXLY_DEFAULT_DEPTH", value: "1.5"},
		{key: "MUXLY_ALWAYS_KILL_ON_LAST_SESSION", value: "maybe"},
	}

	for _, tt := range tests {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)

			_, report := Load(writeConfigFile(t, "entry_dirs:\n  - path: ~/a\n"))

			if report.FileErr != nil {
				t.Errorf("the file is fine, got %v", report.FileErr)
			}
			if report.EnvErr == nil || !strings.Contains(report.EnvErr.Error(), tt.key) || !strings.Contains(report.EnvErr.Error(), tt.value) {
				t.Errorf("EnvErr = %v, want it to name %s and %q", report.EnvErr, tt.key, tt.value)
			}
		})
	}
}

func TestLoadAppliesDefaultsAndTheEnvironmentToABrokenFile(t *testing.T) {
	t.Setenv("MUXLY_EDITOR", "from-env")

	for _, file := range []string{"bad: [\n", "tempaltes: []\n"} {
		cfg, report := Load(writeConfigFile(t, file))

		var parseErr *ParseError
		if !errors.As(report.FileErr, &parseErr) {
			t.Errorf("FileErr = %v, want a ParseError", report.FileErr)
		}
		if cfg.Settings.Editor != "from-env" || cfg.Settings.DefaultDepth != DefaultScanDepth {
			t.Errorf("settings = %+v", cfg.Settings)
		}
	}
}

func TestLoadSaysWhenThereIsNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "gone.yaml")

	_, report := Load(path)

	if !errors.Is(report.FileErr, ErrNotFound) {
		t.Errorf("FileErr = %v", report.FileErr)
	}
	if !strings.Contains(report.FileErr.Error(), path) {
		t.Errorf("%q doesn't say where it looked", report.FileErr)
	}
}

func TestLoadReportsAFileItCannotRead(t *testing.T) {
	_, report := Load(t.TempDir())

	if report.FileErr == nil || errors.Is(report.FileErr, ErrNotFound) {
		t.Errorf("FileErr = %v", report.FileErr)
	}
}

func TestLoadWarningsNameTheFile(t *testing.T) {
	path := writeConfigFile(t, "entry_dirs:\n  - path: ~/a\nsettings:\n  tmux_base: 1\n")

	_, report := Load(path)

	if err := report.Err(); err != nil {
		t.Fatal(err)
	}
	if len(report.Warnings) != 1 || !strings.HasPrefix(report.Warnings[0], path+": ") || !strings.Contains(report.Warnings[0], "line 4") {
		t.Errorf("warnings = %q", report.Warnings)
	}
}

func TestDefaultPath(t *testing.T) {
	t.Run("XDG_CONFIG_HOME wins", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "/xdg/$dir")

		got, err := DefaultPath()

		if err != nil || got != "/xdg/$dir/muxly/config.yaml" {
			t.Errorf("got %q, %v", got, err)
		}
	})

	t.Run("the user's config directory otherwise", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "/home/me")
		dir, err := os.UserConfigDir()
		if err != nil {
			t.Fatal(err)
		}

		got, err := DefaultPath()

		if err != nil || got != filepath.Join(dir, "muxly", "config.yaml") {
			t.Errorf("got %q, %v", got, err)
		}
	})

	t.Run("it's an error with nothing to go on", func(t *testing.T) {
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("HOME", "")

		if got, err := DefaultPath(); err == nil {
			t.Errorf("got %q, want an error", got)
		}
	})
}

// Validate has to report everything at once, so fixing a config isn't one error at a time.
func TestValidateReportsEveryProblem(t *testing.T) {
	cfg, _, err := Parse([]byte(`scan_dirs:
  - path: ~/a
    template: nope
    depth: 0
    alias: x
  - path: ~/b
    alias: x
entry_dirs:
  - path: ~/c
    template: nope2
templates:
  - name: one
    default: true
    windows:
      - name: main
  - name: one
    default: true
  - windows:
      - name: main
`))
	if err != nil {
		t.Fatal(err)
	}

	got := Validate(&cfg)

	if got == nil {
		t.Fatal("want an error")
	}
	for _, want := range []string{
		`scan_dir "~/a" references unknown template "nope"`,
		`scan_dir "~/a" has depth 0`,
		`duplicate alias "x"`,
		`entry_dir "~/c" references unknown template "nope2"`,
		`duplicate template name "one"`,
		`template "one" must have at least one window`,
		"all templates must have a name",
		"only one template can have default: true, found 2",
	} {
		if !strings.Contains(got.Error(), want) {
			t.Errorf("%q is missing %q", got, want)
		}
	}
}

func TestTheRepoFixtureConfigIsValid(t *testing.T) {
	cfg, report := Load(filepath.Join("..", "..", "testdata", "muxly-fixtures", "config.yaml"))

	if err := report.Err(); err != nil {
		t.Fatal(err)
	}
	if err := Validate(&cfg); err != nil {
		t.Error(err)
	}
}
