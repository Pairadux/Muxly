//go:build unix

package cmd

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// brokenConfigs are files muxly can't use, in the ways it can tell apart.
var brokenConfigs = []struct {
	name string
	text string
	want string
}{
	{name: "a syntax error", text: "entry_dirs:\n  - path: /tmp\n bad: [\n", want: "invalid YAML: line"},
	{name: "a misspelled key", text: validConfig + "tempaltes: []\n", want: `unknown key "tempaltes"`},
	{name: "a value of the wrong type", text: validConfig + "  default_depth: lots\n", want: "line "},
}

func TestUnknownKeysAreErrorsWithALineNumber(t *testing.T) {
	c := newCLI(t).withFakes().writeConfig("entry_dirs:\n  - path: /tmp\n    tempalte: x\n" + validTemplates + "  edtior: vi\n")

	got := c.run()

	if got.code == 0 {
		t.Error("want a non-zero exit")
	}
	for _, want := range []string{c.configPath(), `line 3: unknown key "tempalte"`, `line 11: unknown key "edtior"`, "muxly config edit"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("stderr %q doesn't mention %q", got.stderr, want)
		}
	}
	if c.log("fzf") != nil {
		t.Error("the picker opened")
	}
}

// What viper happened to accept is now an error, and a value yaml itself allows still works.
func TestYamlTypesAreStrictButNotFussy(t *testing.T) {
	t.Run("a quoted number is refused", func(t *testing.T) {
		c := newCLI(t).withFakes().writeConfig("entry_dirs:\n  - path: /tmp\n" + validTemplates + "  default_depth: \"3\"\n")

		if got := c.run(); got.code == 0 {
			t.Error("want a non-zero exit")
		}
	})

	t.Run("yes is a bool", func(t *testing.T) {
		c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130")
		c.writeConfig("entry_dirs:\n  - path: /tmp\ntemplates:\n  - name: d\n    default: yes\n    windows:\n      - name: main\n")

		if got := c.run(); got.code != 0 {
			t.Errorf("exit %d, stderr %q", got.code, got.stderr)
		}
	})
}

// A broken config must not stop you repairing it, or learning what's wrong.
func TestABrokenConfigDoesNotLockYouOut(t *testing.T) {
	for _, tt := range brokenConfigs {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("normal commands refuse, naming the problem", func(t *testing.T) {
				for _, args := range [][]string{nil, {"switch"}, {"kill"}, {"config", "list"}, {"config", "add", "entry", "/"}} {
					c := newCLI(t).withFakes().writeConfig(tt.text)

					got := c.run(args...)

					if got.code == 0 || !strings.Contains(got.stderr, tt.want) {
						t.Errorf("muxly %v: exit %d, stderr %q", args, got.code, got.stderr)
					}
					if c.log("fzf") != nil {
						t.Errorf("muxly %v: the picker opened", args)
					}
					if after := c.readConfig(); after != tt.text {
						t.Errorf("muxly %v changed the config:\n%s", args, after)
					}
				}
			})

			t.Run("config edit opens it, and says if it's still broken", func(t *testing.T) {
				c := newCLI(t).withFakeEditors("fake-editor").writeConfig(tt.text).setEnv("EDITOR=fake-editor")

				got := c.run("config", "edit")

				if log := c.log("editor"); !slices.Equal(log, []string{"fake-editor", c.configPath()}) {
					t.Errorf("editor log = %q", log)
				}
				if got.code == 0 || !strings.Contains(got.stderr, tt.want) {
					t.Errorf("exit %d, stderr %q", got.code, got.stderr)
				}
			})

			t.Run("config edit passes once it's fixed", func(t *testing.T) {
				c := newCLI(t).withFakeEditors().writeConfig(tt.text)
				fixed := filepath.Join(c.home, "fixed.yaml")
				c.writeFile(fixed, validConfig, 0o644)
				// An "editor" that overwrites the file it's given, with cp's own path added.
				c.writeFile(filepath.Join(c.bin, "fixer"), "#!/bin/sh\n/bin/cp \"$FIXED\" \"$1\"\n", 0o755)
				c.setEnv("EDITOR=fixer", "FIXED="+fixed)

				got := c.run("config", "edit")

				if got.code != 0 || !strings.Contains(got.stderr, "validated") {
					t.Errorf("exit %d, stderr %q", got.code, got.stderr)
				}
			})

			t.Run("doctor reports it", func(t *testing.T) {
				c := newCLI(t).withFakes().writeConfig(tt.text)

				got := c.run("doctor")

				if got.code != 1 || !strings.Contains(got.stdout, tt.want) {
					t.Errorf("exit %d, stdout:\n%s", got.code, got.stdout)
				}
				if strings.Contains(got.stdout, "no directories") || strings.Contains(got.stderr, "Error:") {
					t.Errorf("unexpected noise, stdout:\n%s\nstderr: %q", got.stdout, got.stderr)
				}
			})

			t.Run("config init won't replace it without --force", func(t *testing.T) {
				c := newCLI(t).writeConfig(tt.text)

				got := c.run("config", "init")

				if got.code == 0 || !strings.Contains(got.stderr, "--force") {
					t.Errorf("exit %d, stderr %q", got.code, got.stderr)
				}
				if after := c.readConfig(); after != tt.text {
					t.Errorf("config changed:\n%s", after)
				}

				if got := c.run("config", "init", "--force"); got.code != 0 {
					t.Fatalf("--force: exit %d, stderr %q", got.code, got.stderr)
				}
				if got := c.run("config", "list"); got.code != 0 {
					t.Errorf("the new config doesn't load: exit %d, stderr %q", got.code, got.stderr)
				}
			})

			t.Run("completion works", func(t *testing.T) {
				c := newCLI(t).writeConfig(tt.text)

				got := c.run("completion", "zsh")

				if got.code != 0 || got.stdout == "" {
					t.Errorf("exit %d, stderr %q", got.code, got.stderr)
				}
			})
		})
	}
}

// settings.editor is still usable from a file with the wrong key in it, but not from one
// that can't be parsed at all, which falls through to the environment.
func TestConfigEditFindsAnEditorInABrokenFile(t *testing.T) {
	c := newCLI(t).withFakeEditors("from-file", "from-env").setEnv("EDITOR=from-env")

	c.writeConfig("tempaltes: []\nsettings:\n  editor: from-file\n")
	c.run("config", "edit")
	if log := c.log("editor"); len(log) == 0 || log[0] != "from-file" {
		t.Errorf("with a misspelled key: editor log = %q", log)
	}

	c.writeConfig("settings:\n  editor: from-file\n bad: [\n")
	c.run("config", "edit")
	if log := c.log("editor"); len(log) < 3 || log[2] != "from-env" {
		t.Errorf("with a syntax error: editor log = %q", log)
	}
}

func TestConfigFlagWithANewFile(t *testing.T) {
	for _, path := range []string{"new.yaml", "sub/dir/new.yaml"} {
		t.Run(path, func(t *testing.T) {
			c := newCLI(t).withFakes()

			got := c.run("--config", path, "config", "init")

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if !strings.Contains(got.stdout, path) {
				t.Errorf("stdout %q doesn't say where", got.stdout)
			}
			if got := c.run("--config", path, "config", "list"); got.code != 0 {
				t.Errorf("the new file doesn't load: exit %d, stderr %q", got.code, got.stderr)
			}
		})
	}
}

func TestConfigFlagWithAMissingFileNamesIt(t *testing.T) {
	c := newCLI(t).withFakes()

	got := c.run("--config", "typo.yaml", "switch")

	if got.code == 0 || !strings.Contains(got.stderr, "no config file found at typo.yaml") {
		t.Errorf("exit %d, stderr %q", got.code, got.stderr)
	}
}

// Everything that stops muxly has to fail doctor too.
func TestDoctorFailsWheneverMuxlyWould(t *testing.T) {
	const dirs = "entry_dirs:\n  - path: /tmp\n"

	tests := []struct {
		name string
		text string
		env  []string
		want []string
	}{
		{name: "an empty file", text: "", want: []string{"no directories configured", "at least one template"}},
		{name: "only comments", text: "# nothing\n", want: []string{"no directories configured", "at least one template"}},
		{name: "no directories", text: validTemplates, want: []string{"no directories configured"}},
		{name: "no templates", text: dirs, want: []string{"at least one template"}},
		{name: "no default", text: dirs + "templates:\n  - name: a\n    windows:\n      - name: main\n", want: []string{"exactly one template"}},
		{
			name: "several problems, all shown",
			text: dirs + "scan_dirs:\n  - path: /tmp\n    template: nope\ntemplates:\n  - name: a\n    default: true\n    windows:\n      - name: main\n  - name: a\n    default: true\n    windows:\n      - name: main\n",
			want: []string{"unknown template", "duplicate template name", "only one template can have default"},
		},
		{name: "a template with no windows", text: dirs + "templates:\n  - name: a\n    default: true\n", want: []string{"must have at least one window"}},
		{name: "a depth below 1", text: "scan_dirs:\n  - path: /tmp\n    depth: 0\n" + validTemplates, want: []string{"must be at least 1"}},
		{name: "a misspelled key", text: validConfig + "tempaltes: []\n", want: []string{`line 10: unknown key "tempaltes"`}},
		{name: "a second document", text: validConfig + "---\nentry_dirs: []\n", want: []string{"a second YAML document"}},
		{name: "a bad depth variable", text: validConfig, env: []string{"MUXLY_DEFAULT_DEPTH=abc"}, want: []string{"MUXLY_DEFAULT_DEPTH"}},
		{name: "a bad kill variable", text: validConfig, env: []string{"MUXLY_ALWAYS_KILL_ON_LAST_SESSION=maybe"}, want: []string{"MUXLY_ALWAYS_KILL_ON_LAST_SESSION"}},
		{name: "two bad variables, both shown", text: validConfig, env: []string{"MUXLY_DEFAULT_DEPTH=abc", "MUXLY_ALWAYS_KILL_ON_LAST_SESSION=maybe"}, want: []string{"MUXLY_DEFAULT_DEPTH", "MUXLY_ALWAYS_KILL_ON_LAST_SESSION"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(tt.text).setEnv(tt.env...)

			muxly := c.run()
			if c.log("fzf") != nil {
				t.Error("the picker opened")
			}
			doctor := c.run("doctor")

			if muxly.code == 0 {
				t.Errorf("muxly: exit 0, want it to refuse")
			}
			if doctor.code != 1 {
				t.Errorf("doctor: exit %d, want 1:\n%s", doctor.code, doctor.stdout)
			}
			for _, want := range tt.want {
				if !strings.Contains(muxly.stderr, want) {
					t.Errorf("muxly's stderr %q doesn't mention %q", muxly.stderr, want)
				}
				if !strings.Contains(doctor.stdout, want) {
					t.Errorf("doctor's stdout doesn't mention %q:\n%s", want, doctor.stdout)
				}
			}
		})
	}

	t.Run("no file at all", func(t *testing.T) {
		c := newCLI(t).withFakes()

		muxly := c.run()
		doctor := c.run("doctor")

		if muxly.code == 0 || !strings.Contains(muxly.stderr, "no config file found") {
			t.Errorf("muxly: exit %d, stderr %q", muxly.code, muxly.stderr)
		}
		if doctor.code != 1 || !strings.Contains(doctor.stdout, "Config file not found") {
			t.Errorf("doctor: exit %d:\n%s", doctor.code, doctor.stdout)
		}
		if strings.Contains(doctor.stdout, "no directories") {
			t.Errorf("doctor complains about what's in a file that isn't there:\n%s", doctor.stdout)
		}
	})

	t.Run("a good config passes both", func(t *testing.T) {
		c := newCLI(t).withFakes().writeConfig(validConfig).setEnv("FAKE_FZF_EXIT=130", "EDITOR=sh")
		c.writeFile(filepath.Join(c.bin, "sh"), "#!/bin/sh\n", 0o755)

		if got := c.run(); got.code != 0 {
			t.Errorf("muxly: exit %d, stderr %q", got.code, got.stderr)
		}
		if got := c.run("doctor"); got.code != 0 || got.stderr != "" {
			t.Errorf("doctor: exit %d, stderr %q:\n%s", got.code, got.stderr, got.stdout)
		}
	})
}

// A misspelled key leaves the rest of the file readable, so doctor can also say what the
// misspelling caused, which is often the clearer of the two.
func TestDoctorShowsAMisspelledKeyAndWhatFollowsFromIt(t *testing.T) {
	c := newCLI(t).withFakes().writeConfig("entry_dirs:\n  - path: /tmp\ntempaltes: []\n")

	got := c.run("doctor")

	if got.code != 1 {
		t.Errorf("exit %d", got.code)
	}
	for _, want := range []string{`line 3: unknown key "tempaltes"`, "at least one template"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("stdout doesn't mention %q:\n%s", want, got.stdout)
		}
	}
}

func TestDoctorFailureIsJustTheSummary(t *testing.T) {
	c := newCLI(t).withFakes()

	got := c.run("doctor")

	if got.code != 1 {
		t.Errorf("exit %d", got.code)
	}
	if got.stderr != "" {
		t.Errorf("stderr %q, want nothing, since the summary said it", got.stderr)
	}
	if !strings.Contains(got.stdout, "error(s)") {
		t.Errorf("stdout:\n%s", got.stdout)
	}
}

func TestTmuxBaseIsAWarningThatNamesTheLine(t *testing.T) {
	const config = validConfig + "  tmux_base: 1\n"

	t.Run("muxly carries on", func(t *testing.T) {
		c := newCLI(t).withFakes().writeConfig(config).setEnv("FAKE_FZF_EXIT=130")

		got := c.run()

		if got.code != 0 {
			t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
		}
		if c.log("fzf") == nil {
			t.Error("the picker didn't open")
		}
		for _, want := range []string{"Warning:", c.configPath(), "tmux_base on line 10", "removed", "delete"} {
			if !strings.Contains(got.stderr, want) {
				t.Errorf("stderr %q doesn't mention %q", got.stderr, want)
			}
		}
	})

	t.Run("it comes before the picker opens, never during", func(t *testing.T) {
		const marker = "PICKER-OPEN"
		c := newCLI(t).withFakes().writeConfig(config)
		c.writeFile(filepath.Join(c.bin, "fzf"), "#!/bin/sh\necho "+marker+" >&2\nexit 130\n", 0o755)

		got := c.run()

		warned, opened := strings.Index(got.stderr, "tmux_base"), strings.Index(got.stderr, marker)
		if warned < 0 || opened < 0 || warned > opened {
			t.Errorf("stderr %q", got.stderr)
		}
	})

	t.Run("doctor warns and passes", func(t *testing.T) {
		c := newCLI(t).withFakes().writeConfig(config)

		got := c.run("doctor")

		if got.code != 0 || !strings.Contains(got.stdout, "tmux_base on line 10") {
			t.Errorf("exit %d:\n%s", got.code, got.stdout)
		}
		if got.stderr != "" {
			t.Errorf("stderr %q, want it reported once, in the checks", got.stderr)
		}
	})

	t.Run("config edit mentions it and passes", func(t *testing.T) {
		c := newCLI(t).withFakeEditors("fake-editor").writeConfig(config).setEnv("EDITOR=fake-editor")

		got := c.run("config", "edit")

		if got.code != 0 || !strings.Contains(got.stderr, "tmux_base on line 10") {
			t.Errorf("exit %d, stderr %q", got.code, got.stderr)
		}
	})

	t.Run("config add leaves it alone", func(t *testing.T) {
		c := newCLI(t).writeConfig(config)

		got := c.run("config", "add", "entry", "~")

		if got.code != 0 {
			t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
		}
		if !strings.Contains(c.readConfig(), "tmux_base: 1") {
			t.Errorf("config:\n%s", c.readConfig())
		}
	})
}

func TestDocumentedEnvironmentVariables(t *testing.T) {
	t.Run("MUXLY_DEFAULT_DEPTH", func(t *testing.T) {
		c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130", "MUXLY_DEFAULT_DEPTH=2")
		c.mkdirs("scan/one/two/three", "pinned/one/two")
		c.writeConfig("scan_dirs:\n  - path: ~/scan\n  - path: ~/pinned\n    depth: 1\n    alias: p\n" + validTemplates)

		if got := c.run(); got.code != 0 {
			t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
		}

		labels := c.pickerLabels()
		// It's the default, so it reaches the deeper folder in ~/scan, where no depth is set,
		// but a scan directory with its own depth keeps it.
		if !slices.Contains(labels, "two") || slices.Contains(labels, "three") || slices.Contains(labels, "p/two") {
			t.Errorf("the picker offered %q", labels)
		}
	})

	t.Run("MUXLY_TMUX_SESSION_PREFIX", func(t *testing.T) {
		c := newCLI(t).writeConfig(validConfig).setEnv("FAKE_FZF_EXIT=130", "MUXLY_TMUX_SESSION_PREFIX=[WORK] ")
		srv := c.withRealTmux()
		srv.NewSession("running")

		if got := c.run(); got.code != 0 {
			t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
		}

		if labels := c.pickerLabels(); !slices.Contains(labels, "[WORK] running") {
			t.Errorf("the picker offered %q", labels)
		}
	})

	t.Run("MUXLY_ALWAYS_KILL_ON_LAST_SESSION", func(t *testing.T) {
		c := newCLI(t).writeConfig(validConfig).setEnv("MUXLY_ALWAYS_KILL_ON_LAST_SESSION=true")
		srv := c.withRealTmux()
		only := srv.NewSession("only")
		c.insideTmux(srv, only)

		got := c.run("kill")

		if got.code != 0 {
			t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
		}
		if srv.HasServer() {
			t.Errorf("the server is still running: %q", srv.Sessions())
		}
	})

	t.Run("MUXLY_EDITOR", func(t *testing.T) {
		c := newCLI(t).withFakeEditors("from-muxly", "from-settings").setEnv("MUXLY_EDITOR=from-muxly")
		c.writeConfig(validConfig + "  editor: from-settings\n")

		if got := c.run("config", "edit"); got.code != 0 {
			t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
		}

		if log := c.log("editor"); len(log) == 0 || log[0] != "from-muxly" {
			t.Errorf("editor log = %q", log)
		}
	})
}

// A "$" or space in the path to the config means itself. Viper read "$" as a variable.
func TestConfigDirectoryCanHoldSpecialCharacters(t *testing.T) {
	c := newCLI(t).withFakes()
	configHome := filepath.Join(c.home, "$dollar and space")
	c.setEnv("XDG_CONFIG_HOME=" + configHome)
	c.writeFile(filepath.Join(configHome, "muxly", "config.yaml"), validConfig, 0o644)

	if got := c.run("config", "list"); got.code != 0 {
		t.Errorf("exit %d, stderr %q", got.code, got.stderr)
	}
}

// Only config.yaml is the config. Viper would also read a config.json or config.yml there,
// so the same setup could mean different things on different days.
func TestOnlyConfigYamlIsRead(t *testing.T) {
	for _, name := range []string{"config.yml", "config.json", "config"} {
		t.Run(name, func(t *testing.T) {
			c := newCLI(t).withFakes()
			c.writeFile(filepath.Join(filepath.Dir(c.configPath()), name), validConfig, 0o644)

			got := c.run("config", "list")

			if got.code == 0 || !strings.Contains(got.stderr, c.configPath()) {
				t.Errorf("exit %d, stderr %q", got.code, got.stderr)
			}
		})
	}
}
