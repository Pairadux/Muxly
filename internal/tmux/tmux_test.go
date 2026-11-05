package tmux

import (
	"errors"
	"reflect"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

func TestBuildWindowArgs(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")

	tests := []struct {
		name        string
		isFirst     bool
		sessionName string
		windowName  string
		dir         string
		cmd         string
		expected    []string
	}{
		{
			name:        "first window no command",
			isFirst:     true,
			sessionName: "myproject",
			windowName:  "main",
			dir:         "/home/user/project",
			cmd:         "",
			expected:    []string{"new-session", "-ds", "myproject", "-n", "main", "-c", "/home/user/project"},
		},
		{
			name:        "subsequent window no command",
			isFirst:     false,
			sessionName: "myproject",
			windowName:  "editor",
			dir:         "/home/user/project",
			cmd:         "",
			expected:    []string{"new-window", "-t", "myproject", "-n", "editor", "-c", "/home/user/project"},
		},
		{
			name:        "first window with command",
			isFirst:     true,
			sessionName: "dev",
			windowName:  "vim",
			dir:         "/home/user/code",
			cmd:         "nvim",
			expected:    []string{"new-session", "-ds", "dev", "-n", "vim", "-c", "/home/user/code", "--", "/bin/zsh", "-lc", "nvim; exec /bin/zsh"},
		},
		{
			name:        "subsequent window with command",
			isFirst:     false,
			sessionName: "dev",
			windowName:  "server",
			dir:         "/home/user/code",
			cmd:         "npm run dev",
			expected:    []string{"new-window", "-t", "dev", "-n", "server", "-c", "/home/user/code", "--", "/bin/zsh", "-lc", "npm run dev; exec /bin/zsh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildWindowArgs(tt.isFirst, tt.sessionName, tt.windowName, tt.dir, tt.cmd)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("buildWindowArgs() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestBuildWindowArgsDefaultShell(t *testing.T) {
	t.Setenv("SHELL", "")

	got := buildWindowArgs(true, "test", "main", "/tmp", "echo hello")

	expectedShell := DefaultShell
	if got[len(got)-3] != expectedShell {
		t.Errorf("buildWindowArgs should use default shell %q when SHELL not set, got %q", expectedShell, got[len(got)-3])
	}
}

func TestGetSessionTarget(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *models.Config
		session  string
		expected string
	}{
		{
			name: "tmux base 1",
			cfg: &models.Config{
				Settings: models.Settings{TmuxBase: 1},
			},
			session:  "myproject",
			expected: "myproject:1",
		},
		{
			name: "tmux base 0",
			cfg: &models.Config{
				Settings: models.Settings{TmuxBase: 0},
			},
			session:  "myproject",
			expected: "myproject:0",
		},
		{
			name: "negative tmux base returns just name",
			cfg: &models.Config{
				Settings: models.Settings{TmuxBase: -1},
			},
			session:  "myproject",
			expected: "myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getSessionTarget(tt.cfg, tt.session)
			if got != tt.expected {
				t.Errorf("getSessionTarget() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCommandErrorMessage(t *testing.T) {
	cause := errors.New("exit status 1")

	tests := []struct {
		name string
		err  commandError
		want string
	}{
		{
			name: "tmux's own message replaces the exit status",
			err:  commandError{subcommand: "kill-session", stderr: "can't find session: x\n", err: cause},
			want: "tmux kill-session: can't find session: x",
		},
		{
			name: "with nothing from tmux, the cause is all there is",
			err:  commandError{subcommand: "kill-session", stderr: " \n", err: cause},
			want: "tmux kill-session: exit status 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
			if !errors.Is(&tt.err, cause) {
				t.Error("the cause should stay reachable through errors.Is")
			}
		})
	}
}
