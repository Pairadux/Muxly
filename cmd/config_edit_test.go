//go:build unix

package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/tmuxtest"
)

func (c *cli) mkdir(path string) string {
	c.t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		c.t.Fatal(err)
	}
	return path
}

// The whole file has to survive, down to the byte: comments, blank lines, unusual
// spacing, quoting, and values muxly could be tempted to rewrite. An editor in the
// environment mustn't end up in the file either.
func TestConfigAddChangesNothingElse(t *testing.T) {
	const before = `# My config

scan_dirs:
    # the main one
    - path: ~/Dev   # keep this note
      depth:   2

entry_dirs:
    - path: "~/notes"

    # - path: ~/old

ignore_dirs: [ target, 'build' ]

templates:
  - name: default
    label:   'Quoted label'
    default: yes
    windows:
      - {name: main, cmd: "echo hi"}

settings:
    default_depth: 3
    tmux_session_prefix: '[T] '
`
	tests := []struct {
		kind   string
		before string
		after  string
	}{
		{kind: "entry", before: "    - path: \"~/notes\"\n", after: "    - path: \"~/notes\"\n    - path: ~/project\n"},
		{kind: "scan", before: "      depth:   2\n", after: "      depth:   2\n    - path: ~/project\n"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			c := newCLI(t).writeConfig(before).setEnv("EDITOR=leaked-from-env", "MUXLY_EDITOR=leaked-too", "VISUAL=leaked-as-well")
			c.mkdir(filepath.Join(c.home, "project"))

			got := c.run("config", "add", tt.kind, "~/project")

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			want := strings.Replace(before, tt.before, tt.after, 1)
			if config := c.readConfig(); config != want {
				t.Errorf("got:\n%s\nwant:\n%s", config, want)
			}
		})
	}
}

func TestConfigAddFlagsAreWritten(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	c.mkdir(filepath.Join(c.home, "dev"))

	got := c.run("config", "add", "scan", "~/dev", "--depth=2", "--alias=d")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	want := "scan_dirs:\n  - path: ~/dev\n    depth: 2\n    alias: d\n"
	if config := c.readConfig(); !strings.Contains(config, want) {
		t.Errorf("config doesn't contain %q:\n%s", want, config)
	}
}

// Paths on the command line start from the current folder, as they would in a shell, and
// go in the file with "~" when they're in the home directory.
func TestConfigAddResolvesPathsFromTheCurrentFolder(t *testing.T) {
	tests := []struct {
		name string
		cwd  string
		arg  string
		want string
	}{
		{name: "dot slash", cwd: "sub", arg: "./foo", want: "~/sub/foo"},
		{name: "bare name", cwd: "sub", arg: "foo", want: "~/sub/foo"},
		{name: "bare path", cwd: ".", arg: "sub/foo", want: "~/sub/foo"},
		{name: "dot", cwd: "sub/foo", arg: ".", want: "~/sub/foo"},
		{name: "dot dot", cwd: "sub/foo/bar", arg: "..", want: "~/sub/foo"},
		{name: "up and over", cwd: "sub/other", arg: "../foo", want: "~/sub/foo"},
		{name: "tilde", cwd: "sub", arg: "~/sub/foo", want: "~/sub/foo"},
		{name: "home itself", cwd: "sub", arg: "~", want: `"~"`},
		{name: "trailing slash", cwd: "sub", arg: "foo/", want: "~/sub/foo"},
	}

	for _, kind := range []string{"entry", "scan"} {
		for _, tt := range tests {
			t.Run(kind+" "+tt.name, func(t *testing.T) {
				c := newCLI(t).writeConfig(validConfig)
				for _, dir := range []string{"sub/foo/bar", "sub/other"} {
					c.mkdir(filepath.Join(c.home, dir))
				}
				c.cwd = filepath.Join(c.home, tt.cwd)
				got := c.run("config", "add", kind, tt.arg)

				if got.code != 0 {
					t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
				}
				if config := c.readConfig(); !strings.Contains(config, "- path: "+tt.want+"\n") {
					t.Errorf("config doesn't list %s:\n%s", tt.want, config)
				}
			})
		}
	}
}

func TestConfigAddOutsideHomeIsAbsolute(t *testing.T) {
	c := newCLI(t).writeConfig(validConfig)
	outside := c.mkdir(filepath.Join(filepath.Dir(c.home), "outside"))

	got := c.run("config", "add", "entry", outside)

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if config := c.readConfig(); !strings.Contains(config, "- path: "+outside+"\n") {
		t.Errorf("config doesn't list %s:\n%s", outside, config)
	}
}

func TestConfigAddSpotsADuplicateHoweverItIsWritten(t *testing.T) {
	for i, written := range []string{"~/notes", "~/notes/", "notes", "$HOME/notes"} {
		for _, kind := range []string{"entry", "scan"} {
			t.Run(kind+strconv.Itoa(i), func(t *testing.T) {
				c := newCLI(t)
				notes := c.mkdir(filepath.Join(c.home, "notes"))
				config := map[string]string{"entry": "entry_dirs", "scan": "scan_dirs"}[kind] + ":\n  - path: " + written + "\n" + validTemplates
				c.writeConfig(config)

				for _, arg := range []string{"~/notes", notes, "./notes"} {
					got := c.run("config", "add", kind, arg)

					if got.code != 0 || !strings.Contains(got.stdout, "already in") {
						t.Errorf("adding %s: exit %d, stdout %q, stderr %q", arg, got.code, got.stdout, got.stderr)
					}
				}
				if after := c.readConfig(); after != config {
					t.Errorf("config changed:\n%s", after)
				}
			})
		}
	}
}

// Bare paths in the file are measured from home, whatever the current folder is. Only
// what's typed on the command line starts from the current folder.
func TestBarePathsInTheConfigStillMeanHome(t *testing.T) {
	c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130")
	c.mkdir(filepath.Join(c.home, "notes"))
	c.cwd = c.mkdir(filepath.Join(c.home, "elsewhere"))
	here := c.mkdir(filepath.Join(c.cwd, "notes"))
	c.writeConfig("entry_dirs:\n  - path: notes\n" + validTemplates)

	if got := c.run(); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if labels := c.pickerLabels(); len(labels) != 1 || labels[0] != "notes" {
		t.Errorf("the picker offered %q", labels)
	}

	got := c.run("config", "add", "entry", "notes")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if strings.Contains(got.stdout, "already in") {
		t.Errorf("./notes was mistaken for ~/notes: %q", got.stdout)
	}
	if config := c.readConfig(); !strings.Contains(config, "- path: notes\n") || !strings.Contains(config, "- path: "+strings.Replace(here, c.home, "~", 1)+"\n") {
		t.Errorf("config:\n%s", config)
	}
}

func TestConfigAddRefusesAPathTheConfigWouldMisread(t *testing.T) {
	for i, name := range []string{"dollar$x", `back\ slash`} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			c := newCLI(t).writeConfig(validConfig)
			dir := c.mkdir(filepath.Join(c.home, name))

			got := c.run("config", "add", "entry", dir)

			if got.code == 0 || !strings.Contains(got.stderr, "can't be listed in the config") {
				t.Errorf("exit %d, stderr %q", got.code, got.stderr)
			}
			if config := c.readConfig(); config != validConfig {
				t.Errorf("config changed:\n%s", config)
			}
		})
	}
}

func TestConfigRemoveFindsAPathHoweverItIsGiven(t *testing.T) {
	tests := []struct {
		name    string
		written string
		arg     string
	}{
		{name: "tilde for tilde", written: "~/notes", arg: "~/notes"},
		{name: "absolute for tilde", written: "~/notes", arg: "<home>/notes"},
		{name: "relative for tilde", written: "~/notes", arg: "./notes"},
		{name: "bare for tilde", written: "~/notes", arg: "notes"},
		{name: "as listed, a bare path from home", written: "sub/notes", arg: "sub/notes"},
		{name: "as listed, a variable", written: "$HOME/notes", arg: "$HOME/notes"},
		{name: "tilde for a variable", written: "$HOME/notes", arg: "~/notes"},
		{name: "as listed, one that can't resolve", written: "~other/x", arg: "~other/x"},
	}

	for _, kind := range []string{"entry", "scan"} {
		for _, tt := range tests {
			t.Run(kind+" "+tt.name, func(t *testing.T) {
				c := newCLI(t)
				c.mkdir(filepath.Join(c.home, "notes"))
				list := map[string]string{"entry": "entry_dirs", "scan": "scan_dirs"}[kind]
				c.writeConfig(list + ":\n  - path: " + tt.written + "\n  - path: /keep\n" + validTemplates)

				got := c.run("config", "remove", kind, strings.Replace(tt.arg, "<home>", c.home, 1))

				if got.code != 0 {
					t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
				}
				want := list + ":\n  - path: /keep\n" + validTemplates
				if config := c.readConfig(); config != want {
					t.Errorf("got:\n%s\nwant:\n%s", config, want)
				}
				if !strings.Contains(got.stdout, tt.written) {
					t.Errorf("stdout %q doesn't say what was removed", got.stdout)
				}
			})
		}
	}
}

// With nothing left, every command but "config edit" refuses to run. Nothing else may
// happen either: the .muxly file only goes once the entry has.
func TestConfigRemoveRefusesTheLastDirectory(t *testing.T) {
	t.Run("entry", func(t *testing.T) {
		c := newCLI(t)
		proj := c.mkdir(filepath.Join(c.home, "proj"))
		muxlyFile := filepath.Join(proj, ".muxly")
		c.writeFile(muxlyFile, "windows:\n  - name: main\n", 0o644)
		config := "entry_dirs:\n  - path: ~/proj\n" + validTemplates
		c.writeConfig(config)

		got := c.run("config", "remove", "entry", "--delete", proj)

		if got.code == 0 || !strings.Contains(got.stderr, "no directories configured") {
			t.Errorf("exit %d, stderr %q", got.code, got.stderr)
		}
		if after := c.readConfig(); after != config {
			t.Errorf("config changed:\n%s", after)
		}
		if _, err := os.Stat(muxlyFile); err != nil {
			t.Errorf(".muxly is gone: %v", err)
		}
	})

	t.Run("scan", func(t *testing.T) {
		c := newCLI(t)
		config := "scan_dirs:\n  - path: ~/dev\n" + validTemplates
		c.writeConfig(config)

		got := c.run("config", "remove", "scan", "~/dev")

		if got.code == 0 || !strings.Contains(got.stderr, "no directories configured") {
			t.Errorf("exit %d, stderr %q", got.code, got.stderr)
		}
		if after := c.readConfig(); after != config {
			t.Errorf("config changed:\n%s", after)
		}
	})
}

func TestConfigAddFollowsASymlinkedConfig(t *testing.T) {
	c := newCLI(t)
	c.mkdir(filepath.Join(c.home, "dev"))
	dotfile := filepath.Join(c.home, "dotfiles", "muxly.yaml")
	c.writeFile(dotfile, validConfig, 0o600)
	c.mkdir(filepath.Dir(c.configPath()))
	if err := os.Symlink(dotfile, c.configPath()); err != nil {
		t.Fatal(err)
	}

	got := c.run("config", "add", "scan", "~/dev")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	info, err := os.Lstat(c.configPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("the config is no longer a symlink")
	}
	data, err := os.ReadFile(dotfile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "~/dev") {
		t.Errorf("the dotfile wasn't changed:\n%s", data)
	}
	dotfileInfo, err := os.Stat(dotfile)
	if err != nil {
		t.Fatal(err)
	}
	if dotfileInfo.Mode().Perm() != 0o600 {
		t.Errorf("mode = %o", dotfileInfo.Mode().Perm())
	}
}

// withFakeEditors installs commands that record how they were called, one argument per
// line. "sh" goes on PATH too, since that's what runs them.
func (c *cli) withFakeEditors(names ...string) *cli {
	c.t.Helper()
	if err := os.Symlink(tmuxtest.Require(c.t, "sh"), filepath.Join(c.bin, "sh")); err != nil {
		c.t.Fatal(err)
	}
	for _, name := range names {
		script := "#!/bin/sh\nPATH=/usr/bin:/bin\nprintf '%s\\n' " + name + " \"$@\" >> \"$FAKE_LOG_DIR/editor\"\n"
		c.writeFile(filepath.Join(c.bin, name), script, 0o755)
	}
	return c
}

func TestConfigEditPicksTheEditorInOrder(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		config string
		env    []string
		want   string
	}{
		{
			name:   "an argument beats everything",
			args:   []string{"from-arg"},
			config: validConfig + "  editor: from-settings\n",
			env:    []string{"MUXLY_EDITOR=from-muxly", "VISUAL=from-visual", "EDITOR=from-editor"},
			want:   "from-arg",
		},
		{
			name:   "MUXLY_EDITOR beats settings",
			config: validConfig + "  editor: from-settings\n",
			env:    []string{"MUXLY_EDITOR=from-muxly", "VISUAL=from-visual", "EDITOR=from-editor"},
			want:   "from-muxly",
		},
		{
			name:   "settings beat VISUAL and EDITOR",
			config: validConfig + "  editor: from-settings\n",
			env:    []string{"VISUAL=from-visual", "EDITOR=from-editor"},
			want:   "from-settings",
		},
		{
			name:   "VISUAL beats EDITOR",
			config: validConfig,
			env:    []string{"VISUAL=from-visual", "EDITOR=from-editor"},
			want:   "from-visual",
		},
		{name: "EDITOR", config: validConfig, env: []string{"EDITOR=from-editor"}, want: "from-editor"},
		{name: "vi if nothing is set", config: validConfig, want: "vi"},
		{name: "a blank argument doesn't count", args: []string{" "}, config: validConfig, env: []string{"EDITOR=from-editor"}, want: "from-editor"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t).withFakeEditors("from-arg", "from-muxly", "from-settings", "from-visual", "from-editor", "vi")
			c.writeConfig(tt.config).setEnv(tt.env...)

			got := c.run(append([]string{"config", "edit"}, tt.args...)...)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			want := []string{tt.want, c.configPath()}
			if log := c.log("editor"); !slices.Equal(log, want) {
				t.Errorf("editor log = %q, want %q", log, want)
			}
		})
	}
}

// An editor is a command line, not just a program, and the path is one argument whatever
// it contains.
func TestConfigEditEditorCanHaveArguments(t *testing.T) {
	c := newCLI(t).withFakeEditors("code").setEnv("EDITOR=code --wait --new-window")
	path := filepath.Join(c.home, "my config; touch oops.yaml")
	c.writeFile(path, validConfig, 0o644)

	got := c.run("--config", path, "config", "edit")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	want := []string{"code", "--wait", "--new-window", path}
	if log := c.log("editor"); !slices.Equal(log, want) {
		t.Errorf("editor log = %q, want %q", log, want)
	}
}

func TestConfigEditReportsAMissingEditor(t *testing.T) {
	c := newCLI(t).withFakeEditors().writeConfig(validConfig).setEnv("EDITOR=not-installed")

	got := c.run("config", "edit")

	if got.code == 0 || !strings.Contains(got.stderr, "not-installed") {
		t.Errorf("exit %d, stderr %q: want the shell's message about the editor", got.code, got.stderr)
	}
}

func TestNewConfigsLeaveTheEditorToTheEnvironment(t *testing.T) {
	c := newCLI(t).withFakeEditors("from-editor").setEnv("EDITOR=from-editor")

	if got := c.run("config", "init"); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	for line := range strings.Lines(c.readConfig()) {
		if strings.HasPrefix(strings.TrimSpace(line), "editor:") {
			t.Errorf("the new config sets an editor: %q", line)
		}
	}

	if got := c.run("config", "edit"); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if log := c.log("editor"); len(log) == 0 || log[0] != "from-editor" {
		t.Errorf("editor log = %q", log)
	}
}

func TestDoctorReportsTheEditorThatWouldRun(t *testing.T) {
	c := newCLI(t).withFakes().withFakeEditors("from-settings", "from-editor")
	c.writeConfig(validConfig + "  editor: from-settings --wait\n").setEnv("EDITOR=from-editor")

	got := c.run("doctor")

	if !strings.Contains(got.stdout, "from-settings --wait") || strings.Contains(got.stdout, "from-editor") {
		t.Errorf("exit %d, stdout:\n%s", got.code, got.stdout)
	}
}

func TestShortFormEntryDirs(t *testing.T) {
	c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130")
	c.mkdirs("notes", "docs", "new")
	c.writeConfig("entry_dirs:\n  - ~/notes\n  - path: ~/docs\n" + validTemplates)

	if got := c.run(); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if labels, want := c.pickerLabels(), []string{"docs", "notes"}; !slices.Equal(labels, want) {
		t.Errorf("the picker offered %q, want %q", labels, want)
	}

	if got := c.run("config", "list"); !strings.Contains(got.stdout, "  ~/notes\n") {
		t.Errorf("config list: exit %d, stdout %q", got.code, got.stdout)
	}

	if got := c.run("config", "add", "entry", "~/notes"); !strings.Contains(got.stdout, "already in") {
		t.Errorf("adding it again: exit %d, stdout %q, stderr %q", got.code, got.stdout, got.stderr)
	}
	if got := c.run("config", "add", "entry", "~/new"); got.code != 0 {
		t.Errorf("adding another: exit %d, stderr %q", got.code, got.stderr)
	}
	if got := c.run("config", "remove", "entry", "~/notes"); got.code != 0 {
		t.Errorf("removing it: exit %d, stderr %q", got.code, got.stderr)
	}

	want := "entry_dirs:\n  - path: ~/docs\n  - path: ~/new\n" + validTemplates
	if config := c.readConfig(); config != want {
		t.Errorf("got:\n%s\nwant:\n%s", config, want)
	}
}

// README's own example writes "- path: ~", which yaml reads as null.
func TestConfigListShowsAnUnquotedTilde(t *testing.T) {
	c := newCLI(t).writeConfig("entry_dirs:\n  - path: ~\n" + validTemplates)

	got := c.run("config", "list")

	if got.code != 0 || !strings.Contains(got.stdout, "entry_dirs:\n  ~\n") {
		t.Errorf("exit %d, stdout %q", got.code, got.stdout)
	}
}
