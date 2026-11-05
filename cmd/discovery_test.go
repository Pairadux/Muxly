//go:build unix

package cmd

import (
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
