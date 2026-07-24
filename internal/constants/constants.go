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

	// Command annotation keys.
	// These mark cobra commands so root's PersistentPreRunE can skip startup
	// checks a given command does not need. Presence of the key is what matters;
	// the value is informational.
	AnnotationSkipUtilsCheck  = "muxly/skip_utils_check"
	AnnotationSkipConfigCheck = "muxly/skip_config_check"
)
