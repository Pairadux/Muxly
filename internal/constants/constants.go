package constants

const (
	// File permissions
	DirectoryPermissions = 0o755
	FilePermissions      = 0o644

	// Channel buffer sizes
	DefaultChannelBufferSize = 100

	// Exit codes
	FzfUserCancelExitCode = 130 // SIGINT (Ctrl+C)

	// Conflict resolution
	MaxConflictResolutionDepth = 10

	// Environment variables
	EnvTmux          = "TMUX"
	EnvShell         = "SHELL"
	EnvXdgConfigHome = "XDG_CONFIG_HOME"
	EnvEditor        = "EDITOR"

	// Common strings
	UserCancelledMsg = "user cancelled"
)
