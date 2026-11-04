//go:build unix

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const existingConfig = "# precious\n" + validConfig

func TestConfigInitCreatesConfigAndSaysWhere(t *testing.T) {
	c := newCLI(t)

	got := c.run("config", "init")

	if got.code != 0 {
		t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
	}
	if !strings.Contains(got.stdout, c.configPath()) {
		t.Errorf("stdout %q doesn't name %s", got.stdout, c.configPath())
	}
	if cfg := c.readConfig(); !strings.Contains(cfg, "templates:") {
		t.Errorf("unexpected config:\n%s", cfg)
	}
	if got := c.run("config", "list"); got.code != 0 {
		t.Errorf("the new config doesn't load: exit %d, stderr %q", got.code, got.stderr)
	}
}

func TestConfigInitRefusesExistingFile(t *testing.T) {
	c := newCLI(t).writeConfig(existingConfig)

	got := c.run("config", "init")

	if got.code == 0 {
		t.Error("want a non-zero exit")
	}
	if !strings.Contains(got.stderr, c.configPath()) || !strings.Contains(got.stderr, "--force") {
		t.Errorf("stderr %q should name the file and --force", got.stderr)
	}
	if got.stdout != "" {
		t.Errorf("stdout = %q, want nothing", got.stdout)
	}
	if cfg := c.readConfig(); cfg != existingConfig {
		t.Errorf("config changed:\n%s", cfg)
	}
}

func TestConfigInitForceOverwrites(t *testing.T) {
	for _, flag := range []string{"--force", "-f"} {
		t.Run(flag, func(t *testing.T) {
			c := newCLI(t).writeConfig(existingConfig)

			got := c.run("config", "init", flag)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			if !strings.Contains(got.stdout, c.configPath()) {
				t.Errorf("stdout %q doesn't name %s", got.stdout, c.configPath())
			}
			if cfg := c.readConfig(); strings.Contains(cfg, "precious") || !strings.Contains(cfg, "templates:") {
				t.Errorf("config wasn't replaced:\n%s", cfg)
			}
		})
	}
}

func TestConfigInitDefaultsFlagIsGone(t *testing.T) {
	for _, flag := range []string{"--Defaults", "--Defaults=false", "-D"} {
		t.Run(flag, func(t *testing.T) {
			c := newCLI(t).writeConfig(existingConfig)

			got := c.run("config", "init", flag)

			if got.code == 0 || !strings.Contains(got.stderr, "unknown") {
				t.Errorf("exit %d, stderr %q: want an unknown flag error", got.code, got.stderr)
			}
			if cfg := c.readConfig(); cfg != existingConfig {
				t.Errorf("config changed:\n%s", cfg)
			}
		})
	}
}

func TestCurrentFolderIsNeverSearchedForConfig(t *testing.T) {
	newProject := func(t *testing.T) (*cli, string) {
		t.Helper()
		c := newCLI(t).withFakes()
		c.cwd = filepath.Join(c.home, "project")
		local := filepath.Join(c.cwd, "config.yaml")
		c.writeFile(local, validConfig, 0o644)
		return c, local
	}

	t.Run("config init leaves it alone", func(t *testing.T) {
		c, local := newProject(t)

		if got := c.run("config", "init"); got.code != 0 {
			t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
		}

		data, err := os.ReadFile(local)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != validConfig {
			t.Errorf("./config.yaml changed:\n%s", data)
		}
		if _, err := os.Stat(c.configPath()); err != nil {
			t.Errorf("global config wasn't created: %v", err)
		}
	})

	t.Run("it isn't loaded", func(t *testing.T) {
		c, _ := newProject(t)

		got := c.run("switch")

		if got.code == 0 || !strings.Contains(got.stderr, "no config file found") {
			t.Errorf("exit %d, stderr %q", got.code, got.stderr)
		}
	})
}

func TestConfigList(t *testing.T) {
	for _, args := range [][]string{{"config", "list"}, {"config", "l"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			c := newCLI(t).writeConfig("scan_dirs:\n  - path: ~/Dev\n    depth: 2\nentry_dirs:\n  - path: ~/notes\nignore_dirs:\n  - target\n" + validTemplates)

			got := c.run(args...)

			if got.code != 0 {
				t.Fatalf("exit %d, stderr %q", got.code, got.stderr)
			}
			want := "scan_dirs:\n  ~/Dev (depth 2)\nentry_dirs:\n  ~/notes\nignore_dirs:\n  target\n"
			if got.stdout != want {
				t.Errorf("stdout:\n%s\nwant:\n%s", got.stdout, want)
			}
		})
	}
}

func TestConfigListNeedsAConfig(t *testing.T) {
	got := newCLI(t).run("config", "list")

	if got.code == 0 || !strings.Contains(got.stderr, "no config file found") {
		t.Errorf("exit %d, stderr %q", got.code, got.stderr)
	}
}
