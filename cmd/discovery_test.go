//go:build unix

package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Pairadux/muxly/internal/tmuxtest"
)

// fastwalk reads a depth of 0 or less as no limit at all, so these must never reach it.
func TestDepthBelowOneIsRejected(t *testing.T) {
	for _, depth := range []string{"0", "-1"} {
		t.Run("root flag "+depth, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)

			got := c.run("--depth=" + depth)

			if got.code == 0 || !strings.Contains(got.stderr, "--depth must be at least 1") {
				t.Errorf("exit %d, stderr %q: want a depth error", got.code, got.stderr)
			}
			if c.log("fzf") != nil {
				t.Error("the picker opened")
			}
		})

		t.Run("config add scan flag "+depth, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)
			dir := c.mkdirs("dev")[0]

			got := c.run("config", "add", "scan", dir, "--depth="+depth)

			if got.code == 0 || !strings.Contains(got.stderr, "--depth must be at least 1") {
				t.Errorf("exit %d, stderr %q: want a depth error", got.code, got.stderr)
			}
			if strings.Contains(c.readConfig(), "dev") {
				t.Errorf("the config changed:\n%s", c.readConfig())
			}
		})

		t.Run("config file "+depth, func(t *testing.T) {
			c := newCLI(t).withFakes()
			c.writeConfig("scan_dirs:\n  - path: /tmp\n    depth: " + depth + "\n" + validTemplates)

			got := c.run()

			if got.code == 0 || !strings.Contains(got.stderr, "depth") {
				t.Errorf("exit %d, stderr %q: want a depth error", got.code, got.stderr)
			}
			if c.log("fzf") != nil {
				t.Error("the picker opened")
			}
		})
	}
}

func TestDepthFlagStillWorks(t *testing.T) {
	c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130")
	c.mkdirs("scan/one/two/three")
	c.writeConfig("scan_dirs:\n  - path: " + filepath.Join(c.home, "scan") + "\n" + validTemplates)

	got := c.run("--depth=2")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	labels := c.pickerLabels()
	if !slices.Contains(labels, "two") || slices.Contains(labels, "three") {
		t.Errorf("the picker offered %q", labels)
	}
}

func TestConfigAddRefusesAFile(t *testing.T) {
	for _, kind := range []string{"entry", "scan"} {
		t.Run(kind, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)
			file := filepath.Join(c.home, "notes.txt")
			c.writeFile(file, "not a directory", 0o644)

			got := c.run("config", "add", kind, file)

			if got.code == 0 || !strings.Contains(got.stderr, "is not a directory") {
				t.Errorf("exit %d, stderr %q: want an error about the file", got.code, got.stderr)
			}
			if strings.Contains(c.readConfig(), "notes.txt") {
				t.Errorf("the file was added:\n%s", c.readConfig())
			}
		})
	}
}

func TestConfigAddRefusesAMissingPath(t *testing.T) {
	for _, kind := range []string{"entry", "scan"} {
		t.Run(kind, func(t *testing.T) {
			c := newCLI(t).withFakes().writeConfig(validConfig)

			got := c.run("config", "add", kind, filepath.Join(c.home, "gone"))

			if got.code == 0 || !strings.Contains(got.stderr, "does not exist") {
				t.Errorf("exit %d, stderr %q: want an error about the path", got.code, got.stderr)
			}
		})
	}
}

func TestIgnoredPathsHideWhatIsInsideThem(t *testing.T) {
	c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130")
	c.mkdirs("dev/old/inside/deeper", "dev/keep", "notes/old")
	c.writeConfig(`scan_dirs:
  - path: ~/dev
    depth: 3
entry_dirs:
  - path: ~/dev/old/inside
  - path: ~/notes/old
ignore_dirs:
  - ~/dev/old
` + validTemplates)

	if got := c.run(); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}

	labels := c.pickerLabels()
	for _, ignored := range []string{"inside", "deeper", "dev/old"} {
		if slices.Contains(labels, ignored) {
			t.Errorf("the picker offered %q: %q", ignored, labels)
		}
	}
	if !slices.Contains(labels, "keep") || !slices.Contains(labels, "old") {
		t.Errorf("the picker lost a directory that isn't ignored: %q", labels)
	}
}

// insideShell writes a config and starts a private server with a client in a session
// called "shell", from which each run of muxly switches.
func (c *cli) insideShell(config string) *tmuxtest.Server {
	c.t.Helper()
	c.writeConfig(config)
	srv := c.withRealTmux()
	shell := srv.NewSession("shell")
	srv.Attach(shell)
	c.insideTmux(srv, shell)
	return srv
}

func (c *cli) pick(label string) result {
	c.t.Helper()
	c.setEnv("FAKE_PICK=" + label)
	return c.run()
}

// sessionAt starts a session the way a person might, with no help from muxly.
func sessionAt(srv *tmuxtest.Server, name, path string) {
	srv.Run("new-session", "-d", "-s", name, "-c", path)
}

func TestASessionNamedTheSameFromElsewhereLeavesTheDirectoryAlone(t *testing.T) {
	c := newCLI(t)
	dir := c.mkdirs("dev/foo")[0]
	srv := c.insideShell(configWithEntries(dir))
	sessionAt(srv, "foo", "/tmp")

	got := c.pick("foo")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if labels, want := c.pickerLabels(), []string{"[TMUX] foo", "foo"}; !slices.Equal(labels, want) {
		t.Errorf("rows = %q, want %q", labels, want)
	}
	if sessions, want := srv.Sessions(), []string{"dev-foo", "foo", "shell"}; !slices.Equal(sessions, want) {
		t.Fatalf("sessions = %q, want %q", sessions, want)
	}
	srv.Eventually("the client to reach dev-foo", func() bool { return srv.ClientSession() == "dev-foo" })
	if path := srv.Run("display-message", "-p", "-t", "=dev-foo:", "#{session_path}"); path != dir {
		t.Errorf("dev-foo is at %q, want %q", path, dir)
	}
}

func TestADirectoryIsMatchedByPathHoweverItsSessionIsNamed(t *testing.T) {
	tests := []struct {
		name   string
		suffix string
	}{
		{name: "exact path"},
		{name: "trailing slash", suffix: "/"},
		{name: "dot segments", suffix: "/../backend"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t)
			dir := c.mkdirs("dev/backend")[0]
			srv := c.insideShell(configWithEntries(dir))
			sessionAt(srv, "api", dir+tt.suffix)

			got := c.pick("[TMUX] api (backend)")

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if labels, want := c.pickerLabels(), []string{"[TMUX] api (backend)"}; !slices.Equal(labels, want) {
				t.Errorf("rows = %q, want %q", labels, want)
			}
			srv.Eventually("the client to reach api", func() bool { return srv.ClientSession() == "api" })
			if sessions, want := srv.Sessions(), []string{"api", "shell"}; !slices.Equal(sessions, want) {
				t.Errorf("sessions = %q, want %q", sessions, want)
			}
		})
	}
}

func TestDirectoriesWithTheSameNameGetTheirOwnSessions(t *testing.T) {
	c := newCLI(t)
	dirs := c.mkdirs("Dev/src", "Work/src")
	srv := c.insideShell(configWithEntries(dirs...))

	for _, tt := range []struct{ label, session, path string }{
		{"Dev/src", "dev-src", dirs[0]},
		{"Work/src", "work-src", dirs[1]},
	} {
		if got := c.pick(tt.label); got.code != 0 {
			t.Fatalf("picking %q: exit %d, stderr %q", tt.label, got.code, got.stderr)
		}
		srv.Eventually("the client to arrive", func() bool { return srv.ClientSession() == tt.session })
		if path := srv.Run("display-message", "-p", "-t", "="+tt.session+":", "#{session_path}"); path != tt.path {
			t.Errorf("%s is at %q, want %q", tt.session, path, tt.path)
		}
	}
	if sessions, want := srv.Sessions(), []string{"dev-src", "shell", "work-src"}; !slices.Equal(sessions, want) {
		t.Errorf("sessions = %q, want %q", sessions, want)
	}
}

func TestPickerShowsRealNamesAndAliases(t *testing.T) {
	c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130")
	c.mkdirs(".config", "MyProject", "proj/unique", "my.project")
	c.writeConfig("entry_dirs:\n  - path: ~/.config\n  - path: ~/MyProject\n  - path: ~/my.project\n" +
		"scan_dirs:\n  - path: ~/proj\n    alias: p\n" + validTemplates)

	if got := c.run(); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}

	if labels, want := c.pickerLabels(), []string{".config", "my.project", "MyProject", "p/unique"}; !slices.Equal(labels, want) {
		t.Errorf("rows = %q, want %q", labels, want)
	}
}

func TestAnAliasIsPartOfTheSessionName(t *testing.T) {
	c := newCLI(t)
	dir := c.mkdirs("projects/unique")[0]
	srv := c.insideShell("scan_dirs:\n  - path: " + filepath.Dir(dir) + "\n    alias: proj\n" + validTemplates)

	if got := c.pick("proj/unique"); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}

	srv.Eventually("the client to reach proj-unique", func() bool { return srv.ClientSession() == "proj-unique" })
}

func TestTheCurrentSessionsDirectoryIsHidden(t *testing.T) {
	c := newCLI(t)
	dirs := c.mkdirs("here", "there")
	c.writeConfig(configWithEntries(dirs...)).setEnv("FAKE_FZF_EXIT=130")
	srv := c.withRealTmux()
	current := srv.Run("new-session", "-d", "-P", "-F", "#{session_id}", "-s", "current", "-c", dirs[0])
	c.insideTmux(srv, current)

	if got := c.run(); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}

	if labels, want := c.pickerLabels(), []string{"there"}; !slices.Equal(labels, want) {
		t.Errorf("rows = %q, want %q", labels, want)
	}
}

func TestTemplatesAndMuxlyFilesStillChooseTheLayout(t *testing.T) {
	c := newCLI(t)
	dirs := c.mkdirs("plain", "templated", "filed")
	c.writeFile(filepath.Join(dirs[2], ".muxly"), "windows:\n  - name: from-file\n", 0o644)
	srv := c.insideShell("entry_dirs:\n  - path: " + dirs[0] + "\n  - path: " + dirs[1] + "\n    template: other\n  - path: " + dirs[2] + "\n" +
		"templates:\n  - name: default\n    default: true\n    windows:\n      - name: from-default\n" +
		"  - name: other\n    windows:\n      - name: from-template\n")

	for _, tt := range []struct{ label, window string }{
		{"plain", "from-default"},
		{"templated", "from-template"},
		{"filed", "from-file"},
	} {
		if got := c.pick(tt.label); got.code != 0 {
			t.Fatalf("picking %q: exit %d, stderr %q", tt.label, got.code, got.stderr)
		}
		if window := srv.Run("display-message", "-p", "-t", "="+tt.label+":", "#{window_name}"); window != tt.window {
			t.Errorf("%s has window %q, want %q", tt.label, window, tt.window)
		}
	}
}

// fzf inherits muxly's stderr, so a marker written there shows what came before it opened.
// Looking at the screen can't show that: fzf draws on the alternate screen, which hides
// anything printed earlier, but without one that text would sit above the picker.
func TestScanWarningsComeAfterThePicker(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	const marker = "PICKER-OPEN"
	c := newCLI(t).withFakes()
	c.writeFile(filepath.Join(c.bin, "fzf"), "#!/bin/sh\necho "+marker+" >&2\nexit 130\n", 0o755)
	locked := c.mkdirs("scan/locked")[0]
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(locked, 0o755); err != nil {
			t.Error(err)
		}
	})
	c.writeConfig("scan_dirs:\n  - path: ~/scan\n    depth: 2\n  - path: ~/gone\n" + validTemplates)

	got := c.run()

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	opened := strings.Index(got.stderr, marker)
	if opened < 0 {
		t.Fatalf("the picker never opened: stderr %q", got.stderr)
	}
	for _, warning := range []string{"permission denied", "gone does not exist"} {
		if at := strings.Index(got.stderr, warning); at < opened {
			t.Errorf("want %q after the picker opened, stderr %q", warning, got.stderr)
		}
	}
}

// The same with real fzf on a real terminal, where the warning has to reach the screen once
// the picker closes. Permissions don't stop root, so this can't be tested as root.
func TestScanWarningsAppearWhenThePickerCloses(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	c := newCLI(t)
	dirs := c.mkdirs("scan/locked", "scan/readable")
	if err := os.Chmod(dirs[0], 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(dirs[0], 0o755); err != nil {
			t.Error(err)
		}
	})
	c.writeConfig("scan_dirs:\n  - path: ~/scan\n    depth: 2\n" + validTemplates)
	srv := c.withRealTmux()
	c.withRealFzf()
	shell := srv.NewSession("shell")
	srv.Attach(shell)

	typeInPane(srv, shell, c.installMuxly())

	srv.Eventually("the picker to open", func() bool { return strings.Contains(srv.PaneText(shell), "readable") })
	if text := srv.PaneText(shell); strings.Contains(text, "permission denied") {
		t.Errorf("a warning was printed over the picker:\n%s", text)
	}
	srv.Run("send-keys", "-t", shell, "Escape")
	srv.Eventually("the warning to appear", func() bool { return strings.Contains(srv.PaneText(shell), "permission denied") })
}

func TestAMissingScanDirectoryWarnsWithoutVerbose(t *testing.T) {
	c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130")
	c.mkdirs("real")
	c.writeConfig("scan_dirs:\n  - path: ~/gone\nentry_dirs:\n  - path: ~/real\n" + validTemplates)

	got := c.run()

	if got.code != 0 || !strings.Contains(got.stderr, "gone does not exist") {
		t.Errorf("exit %d, stderr %q: want a warning about gone", got.code, got.stderr)
	}
	if labels := c.pickerLabels(); !slices.Equal(labels, []string{"real"}) {
		t.Errorf("rows = %q, want the rest still offered", labels)
	}
}

// tmux opens a session in the home directory when it can't use the path it's given.
func TestPickingAMissingEntryExplainsAndOpensNothing(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*cli) string
		want  string
	}{
		{
			name:  "gone",
			setup: func(c *cli) string { return filepath.Join(c.home, "gone") },
			want:  "does not exist",
		},
		{
			name: "a file",
			setup: func(c *cli) string {
				file := filepath.Join(c.home, "gone")
				c.writeFile(file, "", 0o644)
				return file
			},
			want: "is not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t)
			kept := c.mkdirs("kept")[0]
			srv := c.insideShell(configWithEntries(tt.setup(c), kept))

			got := c.pick("gone [missing]")

			if got.code == 0 || !strings.Contains(got.stderr, tt.want) || !strings.Contains(got.stderr, "muxly config remove entry") {
				t.Errorf("exit %d, stderr %q: want an explanation with a next step", got.code, got.stderr)
			}
			if labels, want := c.pickerLabels(), []string{"gone [missing]", "kept"}; !slices.Equal(labels, want) {
				t.Errorf("rows = %q, want %q", labels, want)
			}
			if sessions, want := srv.Sessions(), []string{"shell"}; !slices.Equal(sessions, want) {
				t.Errorf("sessions = %q, want %q", sessions, want)
			}
		})
	}
}

// sessionPath returns where a session was started, finding it by its exact name.
func sessionPath(srv *tmuxtest.Server, name string) string {
	return srv.Run("display-message", "-p", "-t", "="+name+":", "#{session_path}")
}

func TestOpeningAPathOpensItThere(t *testing.T) {
	tests := []struct {
		name    string
		arg     string
		wantDir string
		session string
	}{
		{name: "the current folder", arg: ".", wantDir: "work/here", session: "here"},
		{name: "the parent folder", arg: "..", wantDir: "work", session: "work"},
		{name: "below the current folder", arg: "./sub", wantDir: "work/here/sub", session: "sub"},
		{name: "beside the current folder", arg: "../beside", wantDir: "work/beside", session: "beside"},
		{name: "a slash with nothing before it means relative", arg: "sub/deep", wantDir: "work/here/sub/deep", session: "deep"},
		{name: "a trailing slash, as tab completion leaves", arg: "sub/", wantDir: "work/here/sub", session: "sub"},
		{name: "from home", arg: "~/work/beside", wantDir: "work/beside", session: "beside"},
		{name: "a dollar is a real dollar", arg: "./$HOME", wantDir: "work/here/$HOME", session: "home"},
		{name: "a dotted name", arg: "./.hidden.dir", wantDir: "work/here/.hidden.dir", session: "hidden_dir"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t)
			here := c.mkdirs("work/here")[0]
			c.mkdirs("work/beside", "work/here/sub/deep", "work/here/$HOME", "work/here/.hidden.dir")
			c.cwd = here
			srv := c.insideShell(validConfig)

			got := c.run(tt.arg)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			srv.Eventually("the client to arrive", func() bool { return srv.ClientSession() == tt.session })
			if path, want := sessionPath(srv, tt.session), filepath.Join(c.home, tt.wantDir); path != want {
				t.Errorf("the session is at %q, want %q", path, want)
			}
		})
	}
}

func TestOpeningAnAbsolutePath(t *testing.T) {
	c := newCLI(t)
	dir := c.mkdirs("somewhere/else")[0]
	srv := c.insideShell(validConfig)

	if got := c.run(dir); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}

	srv.Eventually("the client to arrive", func() bool { return srv.ClientSession() == "else" })
	if path := sessionPath(srv, "else"); path != dir {
		t.Errorf("the session is at %q, want %q", path, dir)
	}
}

func TestOpeningAPathThatIsNotADirectoryOpensNothing(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{name: "missing", arg: "./gone", want: "does not exist"},
		{name: "missing with a slash in the middle", arg: "not/there", want: "does not exist"},
		{name: "absolute and missing", arg: "/no/such/place", want: "does not exist"},
		{name: "a file", arg: "./a-file", want: "is not a directory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newCLI(t)
			c.writeFile(filepath.Join(c.home, "a-file"), "", 0o644)
			srv := c.insideShell(validConfig)

			got := c.run(tt.arg)

			if got.code == 0 || !strings.Contains(got.stderr, tt.want) {
				t.Errorf("exit %d, stderr %q: want an error mentioning %q", got.code, got.stderr, tt.want)
			}
			if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"shell"}) {
				t.Errorf("sessions = %q, want only shell", sessions)
			}
		})
	}
}

func TestOpeningAPathWithASessionGoesToThatSession(t *testing.T) {
	c := newCLI(t)
	dir := c.mkdirs("dev/backend")[0]
	srv := c.insideShell(validConfig)
	sessionAt(srv, "api", dir+"/")

	for _, attempt := range []string{"first", "second"} {
		got := c.run(dir)

		if got.code != 0 {
			t.Fatalf("%s: exit %d, stderr %q", attempt, got.code, got.stderr)
		}
		srv.Eventually("the client to reach api", func() bool { return srv.ClientSession() == "api" })
		if sessions, want := srv.Sessions(), []string{"api", "shell"}; !slices.Equal(sessions, want) {
			t.Fatalf("%s: sessions = %q, want %q", attempt, sessions, want)
		}
		srv.Run("switch-client", "-t", "=shell")
		srv.Eventually("the client to return", func() bool { return srv.ClientSession() == "shell" })
	}
}

// tmux lists sessions by name, so "aaa" comes first even though it's the newer one.
func TestOpeningAPathWithTwoSessionsGoesToTheOlder(t *testing.T) {
	c := newCLI(t)
	dir := c.mkdirs("dev/shared")[0]
	srv := c.insideShell(validConfig)
	sessionAt(srv, "zzz", dir)
	sessionAt(srv, "aaa", dir)

	if got := c.run(dir); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}

	srv.Eventually("the client to reach zzz", func() bool { return srv.ClientSession() == "zzz" })
}

func TestOpeningAConfiguredPathUsesItsConfig(t *testing.T) {
	c := newCLI(t)
	dir := c.mkdirs("projects/unique")[0]
	srv := c.insideShell("scan_dirs:\n  - path: ~/projects\n    alias: proj\n    template: other\n" +
		"templates:\n  - name: default\n    default: true\n    windows:\n      - name: from-default\n" +
		"  - name: other\n    windows:\n      - name: from-template\n")

	if got := c.run(dir); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}

	srv.Eventually("the client to reach proj-unique", func() bool { return srv.ClientSession() == "proj-unique" })
	if window := srv.Run("display-message", "-p", "-t", "=proj-unique:", "#{window_name}"); window != "from-template" {
		t.Errorf("window = %q, want from-template", window)
	}
}

func TestOpeningADirectoryByName(t *testing.T) {
	tests := []struct {
		arg     string
		rel     string
		session string
	}{
		{arg: "MyProject", rel: "Dev/MyProject", session: "myproject"},
		{arg: "myproject", rel: "Dev/MyProject", session: "myproject"},
		{arg: "MYPROJECT", rel: "Dev/MyProject", session: "myproject"},
		{arg: "hidden", rel: ".hidden", session: "hidden"},
		{arg: ".hidden", rel: ".hidden", session: "hidden"},
		{arg: "Dev/src", rel: "Dev/src", session: "dev-src"},
		{arg: "work-src", rel: "Work/src", session: "work-src"},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			c := newCLI(t)
			c.mkdirs("Dev/MyProject", ".hidden", "Dev/src", "Work/src", "elsewhere")
			c.cwd = filepath.Join(c.home, "elsewhere")
			config := configWithEntries("~/Dev/MyProject", "~/.hidden", "~/Dev/src", "~/Work/src")
			srv := c.insideShell(config)

			for _, attempt := range []string{"first", "second"} {
				if got := c.run(tt.arg); got.code != 0 {
					t.Fatalf("%s: exit %d, stderr %q", attempt, got.code, got.stderr)
				}
				srv.Eventually("the client to arrive", func() bool { return srv.ClientSession() == tt.session })
				if sessions := srv.Sessions(); len(sessions) != 2 {
					t.Fatalf("%s: sessions = %q, want shell and one more", attempt, sessions)
				}
				srv.Run("switch-client", "-t", "=shell")
				srv.Eventually("the client to return", func() bool { return srv.ClientSession() == "shell" })
			}
			if path, want := sessionPath(srv, tt.session), filepath.Join(c.home, tt.rel); path != want {
				t.Errorf("the session is at %q, want %q", path, want)
			}
		})
	}
}

func TestANameMatchingSeveralDirectoriesListsThem(t *testing.T) {
	c := newCLI(t)
	c.mkdirs("Dev/src", "Work/src")
	srv := c.insideShell(configWithEntries("~/Dev/src", "~/Work/src"))

	got := c.run("src")

	for _, want := range []string{"Dev/src", "Work/src"} {
		if got.code == 0 || !strings.Contains(got.stderr, want) {
			t.Errorf("exit %d, stderr %q: want a failure listing %q", got.code, got.stderr, want)
		}
	}
	if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"shell"}) {
		t.Errorf("sessions = %q, want only shell", sessions)
	}
}

// Picking from the list and typing a name can reach different places, on purpose: what's
// in tmux's status bar can always be typed, and a path always means that path.
func TestARunningSessionsExactNameBeatsADirectory(t *testing.T) {
	c := newCLI(t)
	dir := c.mkdirs("dev/foo")[0]
	srv := c.insideShell(configWithEntries(dir))
	sessionAt(srv, "foo", "/tmp")

	if got := c.run("foo"); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	srv.Eventually("the client to reach foo", func() bool { return srv.ClientSession() == "foo" })
	if sessions, want := srv.Sessions(), []string{"foo", "shell"}; !slices.Equal(sessions, want) {
		t.Errorf("sessions = %q, want %q", sessions, want)
	}

	if got := c.run(dir); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	srv.Eventually("the client to reach dev-foo", func() bool { return srv.ClientSession() == "dev-foo" })
}

func TestOpeningAMissingEntryByNameExplains(t *testing.T) {
	c := newCLI(t)
	srv := c.insideShell(configWithEntries(filepath.Join(c.home, "gone")))

	got := c.run("gone")

	if got.code == 0 || !strings.Contains(got.stderr, "does not exist") {
		t.Errorf("exit %d, stderr %q: want an explanation", got.code, got.stderr)
	}
	if sessions := srv.Sessions(); !slices.Equal(sessions, []string{"shell"}) {
		t.Errorf("sessions = %q, want only shell", sessions)
	}
}

func TestAnUnknownNameOpensANewSessionHere(t *testing.T) {
	c := newCLI(t)
	here := c.mkdirs("here")[0]
	c.cwd = here
	marker := filepath.Join(c.home, "ran")
	c.writeFile(filepath.Join(here, ".muxly"), "windows:\n  - name: from-file\n    cmd: touch "+marker+"\n", 0o644)
	srv := c.insideShell(validConfig)

	for _, attempt := range []string{"first", "second"} {
		if got := c.run("scratch"); got.code != 0 {
			t.Fatalf("%s: exit %d, stderr %q", attempt, got.code, got.stderr)
		}
		srv.Eventually("the client to reach scratch", func() bool { return srv.ClientSession() == "scratch" })
		if sessions, want := srv.Sessions(), []string{"scratch", "shell"}; !slices.Equal(sessions, want) {
			t.Fatalf("%s: sessions = %q, want %q", attempt, sessions, want)
		}
		srv.Run("switch-client", "-t", "=shell")
		srv.Eventually("the client to return", func() bool { return srv.ClientSession() == "shell" })
	}

	if path := sessionPath(srv, "scratch"); path != here {
		t.Errorf("scratch is at %q, want %q", path, here)
	}
	if window := srv.Run("display-message", "-p", "-t", "=scratch:", "#{window_name}"); window != "main" {
		t.Errorf("window = %q, want the default template's main, not the .muxly file's", window)
	}
	if _, err := os.Stat(marker); err == nil {
		t.Error("the .muxly file in the current folder ran")
	}
}

func TestDifferentSpellingsOfANameShareASession(t *testing.T) {
	c := newCLI(t)
	srv := c.insideShell(validConfig)

	for _, arg := range []string{"a.b", "a_b", "A.B"} {
		if got := c.run(arg); got.code != 0 {
			t.Fatalf("%s: exit %d, stderr %q", arg, got.code, got.stderr)
		}
		if sessions, want := srv.Sessions(), []string{"a_b", "shell"}; !slices.Equal(sessions, want) {
			t.Fatalf("%s: sessions = %q, want %q", arg, sessions, want)
		}
	}
}

func TestAnEmptyArgumentOpensThePicker(t *testing.T) {
	c := newCLI(t).withFakes().setEnv("FAKE_FZF_EXIT=130").writeConfig(validConfig)

	if got := c.run(""); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if c.log("fzf") == nil {
		t.Error("the picker never opened")
	}
}

// A command's name can't also be a directory's, so a path is the way to reach one.
func TestAPathReachesADirectoryNamedLikeACommand(t *testing.T) {
	c := newCLI(t)
	dir := c.mkdirs("doctor")[0]
	srv := c.insideShell(validConfig)

	if got := c.run("./doctor"); got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}

	srv.Eventually("the client to reach doctor", func() bool { return srv.ClientSession() == "doctor" })
	if path := sessionPath(srv, "doctor"); path != dir {
		t.Errorf("the session is at %q, want %q", path, dir)
	}
}
