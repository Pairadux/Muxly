// Package constants holds values shared across muxly's packages.
package constants

const (
	// DirectoryPermissions is the mode for directories muxly creates.
	DirectoryPermissions = 0o755
	// FilePermissions is the mode for files muxly creates.
	FilePermissions = 0o644

	// DefaultChannelBufferSize sizes the buffer that collects directory scan results.
	DefaultChannelBufferSize = 100

	// FzfUserCancelExitCode is fzf's exit code when the user presses Esc or Ctrl-C.
	FzfUserCancelExitCode = 130

	// EnvTmux is set by tmux inside its panes, so it tells muxly it's running in tmux.
	EnvTmux = "TMUX"
	// EnvShell names the shell that runs window commands.
	EnvShell = "SHELL"
	// EnvXdgConfigHome overrides the directory that holds muxly's config directory.
	EnvXdgConfigHome = "XDG_CONFIG_HOME"

	// AnnotationSkipUtilsCheck marks a cobra command, and everything under it, as not
	// needing tmux or fzf. Root's PersistentPreRunE checks for the key; its value is
	// only informational.
	AnnotationSkipUtilsCheck = "muxly/skip_utils_check"
	// AnnotationSkipConfigCheck marks a cobra command, and everything under it, as not
	// needing a valid config. It works like AnnotationSkipUtilsCheck.
	AnnotationSkipConfigCheck = "muxly/skip_config_check"
)
