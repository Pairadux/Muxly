//go:build unix

package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// runAsMuxlyEnv makes the test binary behave as muxly. CLI tests re-run the binary
// with it set, so they exercise the real command tree, config loading and exit codes
// in a separate process.
const runAsMuxlyEnv = "MUXLY_TEST_RUN_AS_MUXLY"

func TestMain(m *testing.M) {
	if os.Getenv(runAsMuxlyEnv) != "" {
		Execute()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// validTemplates is the part of a valid config that isn't directories.
const validTemplates = `templates:
  - name: default
    default: true
    windows:
      - name: main
settings:
  default_depth: 1
`

// validConfig is a minimal config that passes validation.
const validConfig = "entry_dirs:\n  - path: /tmp\n" + validTemplates

// fakeTmuxScript logs each call and answers the few queries muxly makes. Sessions come
// from FAKE_SESSIONS (comma separated), and an empty list behaves like no server.
const fakeTmuxScript = `#!/bin/sh
PATH=/usr/bin:/bin
printf '%s\n' "$*" >> "$FAKE_LOG_DIR/tmux"
case "$1" in
  list-sessions)
    [ -n "$FAKE_SESSIONS" ] || exit 1
    printf '%s\n' "$FAKE_SESSIONS" | tr ',' '\n'
    ;;
  display-message) printf '%s\n' "$FAKE_CURRENT" ;;
  -V) echo "tmux 3.8" ;;
esac
exit 0
`

// fakeFzfScript logs each call, prints FAKE_PICK and exits with FAKE_FZF_EXIT.
const fakeFzfScript = `#!/bin/sh
PATH=/usr/bin:/bin
cat > "$FAKE_LOG_DIR/fzf"
[ -z "$FAKE_PICK" ] || printf '%s\n' "$FAKE_PICK"
exit "${FAKE_FZF_EXIT:-0}"
`

// cli runs muxly in a sealed environment. PATH holds only the fakes the test asked for,
// HOME is a temp directory, and nothing is inherited from the parent, so a bug can't
// reach a real tmux server or a real config.
type cli struct {
	t      *testing.T
	home   string
	bin    string
	logDir string
	cwd    string
	env    []string
}

func newCLI(t *testing.T) *cli {
	t.Helper()
	root := t.TempDir()
	c := &cli{
		t:      t,
		home:   filepath.Join(root, "home"),
		bin:    filepath.Join(root, "bin"),
		logDir: filepath.Join(root, "logs"),
	}
	c.cwd = c.home
	for _, dir := range []string{c.home, c.bin, c.logDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

// withFakes installs logging fake tmux and fzf commands on PATH.
func (c *cli) withFakes() *cli {
	c.t.Helper()
	c.writeFile(filepath.Join(c.bin, "tmux"), fakeTmuxScript, 0o755)
	c.writeFile(filepath.Join(c.bin, "fzf"), fakeFzfScript, 0o755)
	return c
}

func (c *cli) writeFile(path, content string, mode os.FileMode) {
	c.t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		c.t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		c.t.Fatal(err)
	}
}

func (c *cli) configPath() string {
	return filepath.Join(c.home, ".config", "muxly", "config.yaml")
}

func (c *cli) writeConfig(content string) *cli {
	c.t.Helper()
	c.writeFile(c.configPath(), content, 0o644)
	return c
}

func (c *cli) readConfig() string {
	c.t.Helper()
	data, err := os.ReadFile(c.configPath())
	if err != nil {
		c.t.Fatal(err)
	}
	return string(data)
}

// setEnv adds KEY=value pairs for later runs.
func (c *cli) setEnv(pairs ...string) *cli {
	c.env = append(c.env, pairs...)
	return c
}

// log returns the calls a fake recorded, one per line, or nil if it never ran.
func (c *cli) log(fake string) []string {
	c.t.Helper()
	data, err := os.ReadFile(filepath.Join(c.logDir, fake))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		c.t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

// tmuxCalled reports whether a tmux call starting with the given subcommand was made.
func (c *cli) tmuxCalled(subcommand string) bool {
	c.t.Helper()
	for _, call := range c.log("tmux") {
		if call == subcommand || strings.HasPrefix(call, subcommand+" ") {
			return true
		}
	}
	return false
}

type result struct {
	stdout string
	stderr string
	code   int
}

func (c *cli) run(args ...string) result {
	c.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], args...)
	cmd.Dir = c.cwd
	cmd.Env = append([]string{
		runAsMuxlyEnv + "=1",
		"PATH=" + c.bin,
		"HOME=" + c.home,
		"XDG_CONFIG_HOME=" + filepath.Join(c.home, ".config"),
		"FAKE_LOG_DIR=" + c.logDir,
		// With -race, each process otherwise sleeps a second as it exits.
		"GORACE=atexit_sleep_ms=0",
	}, c.env...)
	// A new session has no controlling terminal, so an unexpected prompt fails at once
	// instead of waiting on the developer's real terminal.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			c.t.Fatalf("running muxly %v: %v", args, err)
		}
		code = exitErr.ExitCode()
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), code: code}
}
