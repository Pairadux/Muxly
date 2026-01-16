package cmd

// IMPORTS {{{
import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Pairadux/muxly/internal/config"
	"github.com/Pairadux/muxly/internal/constants"
	"github.com/Pairadux/muxly/internal/fzf"
	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/tmux"
	"github.com/Pairadux/muxly/internal/utility"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
) // }}}

var (
	cfg         models.Config
	cfgFileFlag string
	cfgFilePath string
	verbose     bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "muxly [SESSION]",
	Example: "",
	Short:   "A tool for quickly opening tmux sessions",
	Long:    "A tool for quickly opening tmux sessions\n\nBased on ThePrimeagen's tmux-sessionizer script.",
	Args:    cobra.MaximumNArgs(1),
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error { // {{{
		if isConfigCommand(cmd) {
			return nil
		}

		if err := verifyExternalUtils(); err != nil {
			return err
		}
		if err := validateConfig(); err != nil {
			return err
		}
		warnOnConfigIssues()

		return nil
	}, // }}}
	PreRunE: func(cmd *cobra.Command, args []string) error { // {{{
		if len(args) == 1 {
			switch args[0] {
			case "init":
				return fmt.Errorf("unknown command %q for %q. Did you mean:\n  muxly config init?\n", args[0], cmd.Name())
			case "edit":
				return fmt.Errorf("unknown command %q for %q. Did you mean:\n  muxly config edit?\n", args[0], cmd.Name())
			default:
				return nil
			}
		}
		return nil
	}, // }}}
	RunE: func(cmd *cobra.Command, args []string) error {
		if verbose {
			fmt.Printf("scan_dirs: %v\n", cfg.ScanDirs)
			fmt.Printf("entry_dirs: %v\n", cfg.EntryDirs)
			fmt.Printf("ignore_dirs: %v\n", cfg.IgnoreDirs)
			fmt.Printf("fallback_session: %v\n", cfg.FallbackSession)
			fmt.Printf("tmux_base: %v\n", cfg.TmuxBase)
			fmt.Printf("default_depth: %v\n", cfg.DefaultDepth)
			fmt.Printf("session_layout: %v\n", cfg.SessionLayout)
		}

		flagDepth, _ := cmd.Flags().GetInt("depth")
		entries, err := buildDirectoryEntries(flagDepth)
		if err != nil {
			return fmt.Errorf("failed to build directory entries: %w", err)
		}

		var choiceStr string
		if len(args) == 1 {
			choiceStr = args[0]
		}
		if choiceStr == "" {
			// PERF: Pre-allocate slice with exact capacity to avoid reallocations during append
			names := make([]string, 0, len(entries))
			for name := range entries {
				names = append(names, name)
			}

			slices.SortFunc(names, func(a, b string) int {
				isTmuxA := strings.HasPrefix(a, cfg.TmuxSessionPrefix)
				isTmuxB := strings.HasPrefix(b, cfg.TmuxSessionPrefix)
				if isTmuxA && !isTmuxB {
					return -1
				}
				if !isTmuxA && isTmuxB {
					return 1
				}
				return strings.Compare(strings.ToLower(a), strings.ToLower(b))
			})

			choiceStr, err = fzf.SelectWithFzf(names)
			if err != nil {
				if err.Error() == "user cancelled" {
					return nil
				}
				return fmt.Errorf("selecting with fzf failed: %w", err)
			}

			if choiceStr == "" {
				return nil
			}
		}

		sessionName, _ := strings.CutPrefix(choiceStr, cfg.TmuxSessionPrefix)

		selectedPath, exists := entries[choiceStr]
		if !exists && len(args) == 0 {
			return fmt.Errorf("the name must match an existing directory entry: %s", choiceStr)
		}

		sessionLayout := loadMuxlyFile(selectedPath)
		if len(sessionLayout.Windows) == 0 {
			sessionLayout = cfg.SessionLayout
		}

		session := models.Session{
			Name:   sessionName,
			Path:   selectedPath,
			Layout: sessionLayout,
		}

		if err := tmux.CreateAndSwitchSession(&cfg, session); err != nil {
			return fmt.Errorf("Failed to switch session: %w", err)
		}

		return nil
	},
}

// loadMuxlyFile attempts to load a .muxly file from the given directory.
//
// Returns the parsed SessionLayout if the file exists and is valid YAML,
// or an empty SessionLayout otherwise. This provides project-specific
// session configuration that overrides the global session_layout from config.
//
// Errors are silently ignored since .muxly files are optional overrides.
func loadMuxlyFile(path string) models.SessionLayout {
	layoutPath := filepath.Join(path, ".muxly")

	data, err := os.ReadFile(layoutPath)
	if err != nil {
		return models.SessionLayout{}
	}

	var layout models.SessionLayout
	if err := yaml.Unmarshal(data, &layout); err != nil {
		return models.SessionLayout{}
	}

	return layout
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() { // {{{
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
} // }}}

func init() { // {{{
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFileFlag, "config", "", "config file (default $XDG_CONFIG_HOME/muxly/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.Flags().IntP("depth", "d", 0, "Maximum traversal depth")
} // }}}

// initConfig reads in config file and ENV variables if set.
func initConfig() { // {{{
	if cfgFileFlag != "" {
		cfgFilePath = cfgFileFlag
		viper.SetConfigFile(cfgFilePath)
	} else {
		var configDir string

		xdg_config_home := os.Getenv(constants.EnvXdgConfigHome)
		if xdg_config_home != "" {
			configDir = xdg_config_home
		} else {
			var err error
			configDir, err = os.UserConfigDir()
			if err != nil {
				fmt.Fprintf(os.Stderr, "UserConfigDir cannot be found: %v\n", err)
			}
		}

		cfgDir := filepath.Join(configDir, "muxly")
		viper.AddConfigPath(cfgDir)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
		cfgFilePath = filepath.Join(cfgDir, "config.yaml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// Bind environment variables for config overrides
	// Allows MUXLY_* environment variables to override config file values
	viper.SetEnvPrefix("MUXLY")
	viper.BindEnv("editor", "MUXLY_EDITOR", "EDITOR")           // Support both MUXLY_EDITOR and standard $EDITOR
	viper.BindEnv("default_depth")                              // MUXLY_DEFAULT_DEPTH
	viper.BindEnv("tmux_base")                                  // MUXLY_TMUX_BASE
	viper.BindEnv("tmux_session_prefix")                        // MUXLY_TMUX_SESSION_PREFIX
	viper.BindEnv("always_kill_on_last_session")                // MUXLY_ALWAYS_KILL_ON_LAST_SESSION

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintln(os.Stderr, "Config file is corrupted or unreadable:", err)
			os.Exit(1)
		}
	} else if verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		// FIXME: try unmarshalling 1 key at a time
		fmt.Fprintf(os.Stderr, "Issue unmarshalling config file: %v\n", err)
	}

	// Sync cfgFilePath with the actual config file that was loaded
	// This ensures 'muxly config edit' opens the correct file
	if viper.ConfigFileUsed() != "" {
		cfgFilePath = viper.ConfigFileUsed()
	}
} // }}}

// buildDirectoryEntries creates a map of display names to directory paths by
// processing scan_dirs and entry_dirs from the configuration. It handles
// directory scanning at specified depths, filters out ignored directories,
// excludes the current tmux session, and marks existing tmux sessions with
// a "[TMUX]" prefix.
//
// The flagDepth parameter can override the scanning depth for scan_dirs.
// Returns a map where keys are display names and values are resolved paths
// or session names for existing tmux sessions.
func buildDirectoryEntries(flagDepth int) (map[string]string, error) {
	existingSessions := tmux.GetTmuxSessionSet()
	currentSession := tmux.GetCurrentTmuxSession()

	ignoreSet := buildIgnoreSet()
	allPaths := collectAllPaths(flagDepth, ignoreSet, currentSession)

	entries := make(map[string]string)
	addDirectoryEntries(entries, allPaths, currentSession, existingSessions)
	addTmuxSessionEntries(entries, existingSessions, currentSession)

	return entries, nil
}

// buildIgnoreSet creates a set of resolved paths from cfg.IgnoreDirs for O(1) lookup.
//
// Using a set (map[string]struct{}) instead of a slice allows constant-time checks
// to see if a directory should be ignored, rather than linear-time iteration.
// Paths that fail to resolve are silently skipped.
func buildIgnoreSet() models.StringSet {
	ignoreSet := make(models.StringSet)
	for _, dir := range cfg.IgnoreDirs {
		resolved, err := utility.ResolvePath(dir)
		if err == nil {
			ignoreSet[resolved] = struct{}{}
		}
	}
	return ignoreSet
}

// collectAllPaths gathers all directory paths from scan_dirs and entry_dirs.
//
// For scan_dirs: Recursively scans each directory up to the configured depth,
// respecting the flagDepth override if provided (CLI --depth flag).
//
// For entry_dirs: Adds directories directly without scanning subdirectories.
//
// Directories in ignoreSet and directories matching the current tmux session name
// are filtered out. Each path is tagged with an optional prefix (alias) for display.
//
// Returns a slice of PathInfo structs containing the path and its display prefix.
func collectAllPaths(flagDepth int, ignoreSet models.StringSet, currentSession string) []models.PathInfo {
	var allPaths []models.PathInfo

	addPath := func(path, prefix string) error {
		if _, ignored := ignoreSet[path]; ignored {
			return nil
		}

		name := filepath.Base(path)
		if name == currentSession {
			return nil
		}

		info := models.PathInfo{Path: path, Prefix: prefix}
		allPaths = append(allPaths, info)
		return nil
	}

	for _, scanDir := range cfg.ScanDirs {
		prefix := scanDir.Alias
		if err := processScanDir(scanDir, flagDepth, prefix, addPath); err != nil {
			continue
		}
	}

	for _, entryDir := range cfg.EntryDirs {
		resolved, err := utility.ResolvePath(entryDir)
		if err != nil {
			if verbose {
				fmt.Fprintf(os.Stderr, "Warning: failed to resolve entry directory %s: %v\n", entryDir, err)
			}
			continue
		}
		addPath(resolved, "")
	}

	return allPaths
}

// addDirectoryEntries populates the entries map with display names for directories.
//
// This function handles the complex task of creating unique, user-friendly display names
// for directories that may have the same basename (e.g., multiple "src" directories).
// It calls deduplicateDisplayNames to resolve conflicts by using path suffixes.
//
// Entries that would conflict with existing tmux sessions or match the current session
// are skipped to avoid ambiguity in the selector.
func addDirectoryEntries(entries map[string]string, allPaths []models.PathInfo, currentSession string, existingSessions map[string]bool) {
	displayNames := deduplicateDisplayNames(allPaths)

	for _, info := range allPaths {
		displayName := displayNames[info.Path]

		if shouldSkipEntry(displayName, currentSession, existingSessions) {
			continue
		}

		entries[displayName] = info.Path
	}
}

// addTmuxSessionEntries adds existing tmux sessions to the entries map.
//
// Sessions are prefixed with cfg.TmuxSessionPrefix (default: "[TMUX] ") to distinguish
// them from directory entries in the selector. The current session is excluded since
// you can't switch to the session you're already in.
//
// For these entries, the value is the session name itself (not a path), which tells
// the main logic to switch to an existing session rather than create a new one.
func addTmuxSessionEntries(entries map[string]string, existingSessions map[string]bool, currentSession string) {
	for sessionName := range existingSessions {
		if sessionName == currentSession {
			continue
		}

		displayName := cfg.TmuxSessionPrefix + sessionName
		entries[displayName] = sessionName
	}
}

// processScanDir scans a single scan_dir entry and adds all discovered subdirectories.
//
// Depth priority (highest to lowest):
//  1. CLI flag (--depth)
//  2. Per-directory depth (scanDir.Depth)
//  3. Global default (cfg.DefaultDepth)
//
// This is handled by the ScanDir.GetDepth method. The function resolves the path,
// scans for subdirectories up to the effective depth, and calls addEntry for each.
// Errors are logged if verbose mode is enabled but don't stop execution.
func processScanDir(scanDir models.ScanDir, flagDepth int, prefix string, addEntry func(string, string) error) error {
	defaultDepth := cfg.DefaultDepth
	effectiveDepth := scanDir.GetDepth(flagDepth, defaultDepth)

	// get absolute path
	resolved, err := utility.ResolvePath(scanDir.Path)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to resolve scan directory %s: %v\n", scanDir.Path, err)
		}
		return nil
	}

	// get subdirs
	subDirs, err := utility.GetSubDirs(effectiveDepth, resolved)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to scan directory %s: %v\n", resolved, err)
		}
		return nil
	}

	// add subdirs
	for _, subDir := range subDirs {
		if err := addEntry(subDir, prefix); err != nil {
			return err
		}
	}

	return nil
}

// deduplicateDisplayNames creates unique display names for paths with conflicting basenames.
//
// When multiple paths have the same basename (e.g., ~/Dev/project1/src and ~/Work/project2/src),
// this function finds the minimum path suffix needed to make them distinguishable:
//   - "src" conflicts → try depth 1 → still "src/src" conflicts → try depth 2 → "project1/src" vs "project2/src" ✓
//
// Uses hash-based grouping for O(n) performance instead of O(n²) comparisons.
// Paths without conflicts keep their simple basename. Prefixes (aliases) are applied
// to the final display names.
func deduplicateDisplayNames(allPaths []models.PathInfo) map[string]string {
	if len(allPaths) == 0 {
		return make(map[string]string)
	}

	// Group paths by basename
	groups := make(map[string][]models.PathInfo)
	for _, info := range allPaths {
		basename := filepath.Base(info.Path)
		groups[basename] = append(groups[basename], info)
	}

	result := make(map[string]string)

	// Process each group
	for _, group := range groups {
		if len(group) == 1 {
			// No duplicates, use basename
			info := group[0]
			displayName := normalizeSessionName(filepath.Base(info.Path))
			displayName = applyPrefix(info.Prefix, displayName)
			result[info.Path] = displayName
		} else {
			// Resolve conflicts by finding minimum distinguishing suffix
			resolved := resolveConflicts(group)
			maps.Copy(result, resolved)
		}
	}

	return result
}

// resolveConflicts finds the minimum suffix depth needed to make all paths unique.
//
// Iterates through increasing suffix depths (1, 2, 3, ...) until all paths have unique
// display names. For example, with paths /home/user/Dev/app and /home/user/Work/app:
//   - Depth 1: "app" vs "app" → conflict
//   - Depth 2: "Dev/app" vs "Work/app" → unique ✓
//
// Returns a map of full paths to their unique display names. Caps at maxDepth (10)
// to prevent infinite loops, though this should never happen in practice.
func resolveConflicts(paths []models.PathInfo) map[string]string {
	const maxDepth = 10 // Reasonable limit to prevent infinite loops

	for depth := 1; depth <= maxDepth; depth++ {
		suffixes := make(map[string]models.PathInfo)
		conflicts := false

		for _, info := range paths {
			suffix := getPathSuffix(info.Path, depth)
			if existing, exists := suffixes[suffix]; exists {
				// Check if it's actually the same path (shouldn't happen but safety check)
				if existing.Path != info.Path {
					conflicts = true
					break
				}
			}
			suffixes[suffix] = info
		}

		if !conflicts {
			// All unique at this depth
			result := make(map[string]string)
			for suffix, info := range suffixes {
				displayName := normalizePathForDisplay(suffix)
				displayName = applyPrefix(info.Prefix, displayName)
				result[info.Path] = displayName
			}
			return result
		}
	}

	// Fallback: use full path if conflict cant be resolved
	result := make(map[string]string)
	for _, info := range paths {
		displayName := normalizePathForDisplay(info.Path)
		displayName = applyPrefix(info.Prefix, displayName)
		result[info.Path] = displayName
	}
	return result
}

// getPathSuffix extracts the last N components of a path for display purposes.
//
// Examples:
//   getPathSuffix("/home/user/Dev/my-project", 1) → "my-project"
//   getPathSuffix("/home/user/Dev/my-project", 2) → "Dev/my-project"
//   getPathSuffix("/home/user/Dev/my-project", 5) → "/home/user/Dev/my-project" (entire path)
//
// Used by deduplication logic to create progressively longer display names
// until conflicts are resolved.
func getPathSuffix(path string, depth int) string {
	components := strings.Split(filepath.Clean(path), string(filepath.Separator))

	// Remove empty components (can happen with leading/trailing separators)
	var cleanComponents []string
	for _, comp := range components {
		if comp != "" {
			cleanComponents = append(cleanComponents, comp)
		}
	}

	if depth >= len(cleanComponents) {
		return strings.Join(cleanComponents, string(filepath.Separator))
	}

	start := len(cleanComponents) - depth
	return strings.Join(cleanComponents[start:], string(filepath.Separator))
}

// shouldSkipEntry determines if a directory entry should be excluded from the selector.
//
// Skips entries that would cause ambiguity or confusion:
//   - Matches the current tmux session name (can't switch to yourself)
//   - Conflicts with an existing tmux session name (without the [TMUX] prefix)
//
// This prevents situations where a directory and session have the same name,
// which would make selection ambiguous.
func shouldSkipEntry(displayName, currentSession string, existingSessions map[string]bool) bool {
	return displayName == currentSession || existingSessions[displayName]
}

// isConfigCommand checks if the given command or any of its parent commands
// is "config". This is used to skip config validation for commands like
// "muxly config init" or "muxly config edit", which are intended to manage or
// create the config file.
func isConfigCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "config" {
			return true
		}
	}

	return false
}

// validateConfig ensures that the application configuration is valid and complete.
// It checks for the presence of a config file and validates the config structure.
// Returns an error with helpful instructions if validation fails.
func validateConfig() error {
	if viper.ConfigFileUsed() == "" {
		return fmt.Errorf("no config file found\nRun 'muxly config init' to create one, or use --config to specify a path\n")
	}

	return config.Validate(&cfg)
}

// applyPrefix adds an alias prefix to a display name if one is configured.
//
// Examples:
//   applyPrefix("dev", "my-project") → "dev/my-project"
//   applyPrefix("", "my-project")    → "my-project"
//
// Prefixes come from the scan_dir alias configuration and help organize
// the selector display when you have multiple scan directories.
func applyPrefix(prefix, name string) string {
	if prefix != "" {
		return prefix + "/" + name
	}
	return name
}

// normalizeSessionName converts a directory name into a valid tmux session name.
//
// Tmux session names cannot start with dots, so this function replaces leading dots
// with underscores. For example:
//   ".config" → "_config"
//   ".dotfiles" → "_dotfiles"
//   "regular-name" → "regular-name" (unchanged)
//
// This ensures all directory names can be used as session names without errors.
func normalizeSessionName(name string) string {
	if strings.HasPrefix(name, ".") {
		return "_" + name[1:]
	}
	return name
}

// normalizePathForDisplay normalizes the last component of a path by replacing leading dots with underscores.
// This ensures display names match the actual session names that tmux creates.
func normalizePathForDisplay(path string) string {
	if path == "" {
		return path
	}

	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) > 0 {
		lastIdx := len(parts) - 1
		parts[lastIdx] = normalizeSessionName(parts[lastIdx])
	}
	return strings.Join(parts, string(filepath.Separator))
}

func warnOnConfigIssues() {
	if cfg.Editor == "" {
		fmt.Fprintf(os.Stderr, "Warning: editor not set, defaulting to '%s'\n", config.DefaultEditor)
	}

	if cfg.FallbackSession.Name == "" {
		fmt.Fprintf(os.Stderr, "fallback_session.name is missing, defaulting to '%s'\n", config.DefaultFallbackSessionName)
	}

	if cfg.FallbackSession.Path == "" {
		fmt.Fprintf(os.Stderr, "fallback_session.path is missing, defaulting to '%s'\n", config.DefaultFallbackSessionPath)
	}

	if len(cfg.FallbackSession.Layout.Windows) == 0 {
		fmt.Fprintln(os.Stderr, "fallback_session.layout.windows is empty, using default layout")
	}
}

func verifyExternalUtils() error {
	var missing []string

	if _, err := exec.LookPath("tmux"); err != nil {
		missing = append(missing, "tmux")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		missing = append(missing, "fzf")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required tools: %s", strings.Join(missing, ", "))
	}

	return nil
}
