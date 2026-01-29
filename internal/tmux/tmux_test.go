package tmux

import (
	"os"
	"reflect"
	"testing"

	"github.com/Pairadux/muxly/internal/models"
)

func TestBuildWindowArgs(t *testing.T) {
	originalShell := os.Getenv("SHELL")
	defer os.Setenv("SHELL", originalShell)

	os.Setenv("SHELL", "/bin/zsh")

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
	originalShell := os.Getenv("SHELL")
	os.Unsetenv("SHELL")
	defer os.Setenv("SHELL", originalShell)

	got := buildWindowArgs(true, "test", "main", "/tmp", "echo hello")

	expectedShell := DefaultShell
	if got[len(got)-3] != expectedShell {
		t.Errorf("buildWindowArgs should use default shell %q when SHELL not set, got %q", expectedShell, got[len(got)-3])
	}
}

func TestParseWindows(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected models.SessionLayout
	}{
		{
			name:     "empty input",
			input:    "",
			expected: models.SessionLayout{},
		},
		{
			name:     "whitespace only",
			input:    "   \n\t\n   ",
			expected: models.SessionLayout{},
		},
		{
			name:  "single window no command",
			input: "main",
			expected: models.SessionLayout{
				Windows: []models.Window{
					{Name: "main", Cmd: ""},
				},
			},
		},
		{
			name:  "single window with command",
			input: "editor:nvim",
			expected: models.SessionLayout{
				Windows: []models.Window{
					{Name: "editor", Cmd: "nvim"},
				},
			},
		},
		{
			name:  "multiple windows",
			input: "main\neditor:nvim\nserver:npm run dev",
			expected: models.SessionLayout{
				Windows: []models.Window{
					{Name: "main", Cmd: ""},
					{Name: "editor", Cmd: "nvim"},
					{Name: "server", Cmd: "npm run dev"},
				},
			},
		},
		{
			name:  "handles whitespace",
			input: "  main  \n  editor : nvim  \n  server:npm run dev  ",
			expected: models.SessionLayout{
				Windows: []models.Window{
					{Name: "main", Cmd: ""},
					{Name: "editor", Cmd: "nvim"},
					{Name: "server", Cmd: "npm run dev"},
				},
			},
		},
		{
			name:  "skips blank lines",
			input: "main\n\neditor:nvim\n\n\nserver",
			expected: models.SessionLayout{
				Windows: []models.Window{
					{Name: "main", Cmd: ""},
					{Name: "editor", Cmd: "nvim"},
					{Name: "server", Cmd: ""},
				},
			},
		},
		{
			name:  "command with colons",
			input: "server:docker run -p 8080:80 nginx",
			expected: models.SessionLayout{
				Windows: []models.Window{
					{Name: "server", Cmd: "docker run -p 8080:80 nginx"},
				},
			},
		},
		{
			name:  "skips empty window names",
			input: ":nvim\nmain\n:",
			expected: models.SessionLayout{
				Windows: []models.Window{
					{Name: "main", Cmd: ""},
				},
			},
		},
		{
			name:  "window name with spaces",
			input: "my window:command",
			expected: models.SessionLayout{
				Windows: []models.Window{
					{Name: "my window", Cmd: "command"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseWindows(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseWindows(%q) = %+v, want %+v", tt.input, got, tt.expected)
			}
		})
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
