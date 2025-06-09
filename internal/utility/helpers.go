// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

package utility

// IMPORTS {{{
import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Pairadux/tms/internal/models"

	"github.com/charlievieth/fastwalk"
	"github.com/spf13/viper"
) // }}}

// ResolvePath takes an “unknown” path pattern and returns an absolute path.
//
//   - Absolute (/…):               returned as-is
//   - Home (~ or ~/…):             expanded via os.UserHomeDir()
//   - Explicit relative (./, ../): error
//   - Implicit (foo/bar):          treated as ~/foo/bar
func ResolvePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		switch {
		case p == "~":
			return home, nil
		case strings.HasPrefix(p, "~/"):
			return filepath.Join(home, p[2:]), nil
		default:
			return "", errors.New("invalid home-path syntax")
		}
	}
	if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") || p == "." || p == ".." {
		return "", errors.New("relative paths not allowed: " + p)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, p), nil
}

func GetSubDirs(maxDepth int, root string) ([]string, error) {
	dirChan := make(chan string, 100)
	cfg := &fastwalk.Config{MaxDepth: maxDepth}
	walkFn := func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "walk error %q: %v\n", path, err)
			return nil
		}
		if d.IsDir() {
			dirChan <- path
		}
		return nil
	}
	var dirs []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		for dir := range dirChan {
			dirs = append(dirs, dir)
		}
	}()
	err := fastwalk.Walk(cfg, root, walkFn)
	close(dirChan)
	<-done
	if err != nil {
		return nil, err
	}
	return dirs, nil
}

func GetTmuxSessions() map[string]bool {
	sessions := make(map[string]bool)
	cmd := exec.Command("tmux", "list-sessions", "-F", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return sessions
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			sessions[line] = true
		}
	}
	return sessions
}

func GetCurrentTmuxSession() string {
	if os.Getenv("TMUX") == "" {
		return ""
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func TmuxSwitchSession(name, cwd string) error {
	var sessionLayout models.SessionLayout
	if err := viper.UnmarshalKey("session_layout", &sessionLayout); err != nil {
		return fmt.Errorf("failed to decode session_layout: %w", err)
	}
	sessionExists := exec.Command("tmux", "has-session", "-t", name).Run() == nil
	if !sessionExists {
		if err := CreateSession(sessionLayout, name, cwd); err != nil {
			return fmt.Errorf("creating session: %w", err)
		}
	}
	tmuxBase := viper.GetInt("tmux_base")
	target := fmt.Sprintf("%s:%d", name, tmuxBase)
	if os.Getenv("TMUX") == "" {
		cmd := exec.Command("tmux", "attach-session", "-t", target)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			if target != name {
				cmd := exec.Command("tmux", "attach-session", "-t", name)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
				cmd.Stdin = os.Stdin
				return cmd.Run()
			}
			return fmt.Errorf("attaching to session: %w", err)
		}
	} else {
		if err := exec.Command("tmux", "switch-client", "-t", target).Run(); err != nil {
			if target != name {
				return exec.Command("tmux", "switch-client", "-t", target).Run()
			}
			return fmt.Errorf("switching to session: %w", err)
		}
	}
	return nil
}

func CreateSession(sessionLayout models.SessionLayout, session, dir string) error {
	if len(sessionLayout.Windows) == 0 {
		return fmt.Errorf("no windows defined in session layout")
	}
	w0 := sessionLayout.Windows[0]
	args := []string{"new-session", "-ds", session, "-n", w0.Name, "-c", dir}
	if w0.Cmd != "" {
		args = append(args, strings.Fields(w0.Cmd)...)
	}
	if err := exec.Command("tmux", args...).Run(); err != nil {
		return err
	}
	for _, w := range sessionLayout.Windows[1:] {
		args = []string{"new-window", "-t", session, "-n", w.Name, "-c", dir}
		if w.Cmd != "" {
			args = append(args, w.Cmd)
		}
		if err := exec.Command("tmux", args...).Run(); err != nil {
			return err
		}
	}
	return nil
}

