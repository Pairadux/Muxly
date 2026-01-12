// SPDX-License-Identifier: MIT
// © 2025 Austin Gause <a.gause@outlook.com>

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
			// Build list of all entry names for fzf selection
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

		sessionLayout := retrieveSessionLayout(selectedPath)
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

func retrieveSessionLayout(path string) models.SessionLayout {
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
	viper.BindEnv("editor", "MUXLY_EDITOR", "EDITOR") // Support both MUXLY_EDITOR and standard $EDITOR
	viper.BindEnv("default_depth")                    // MUXLY_DEFAULT_DEPTH
	viper.BindEnv("tmux_base")                        // MUXLY_TMUX_BASE
	viper.BindEnv("tmux_session_prefix")              // MUXLY_TMUX_SESSION_PREFIX
	viper.BindEnv("always_kill_on_last_session")      // MUXLY_ALWAYS_KILL_ON_LAST_SESSION

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

	// Pre-allocate map with capacity for all paths and existing sessions
	entries := make(map[string]string, len(allPaths)+len(existingSessions))
	addDirectoryEntries(entries, allPaths, currentSession, existingSessions)
	addTmuxSessionEntries(entries, existingSessions, currentSession)

	return entries, nil
}

func buildIgnoreSet() models.StringSet {
	// Pre-allocate set with capacity for all ignored directories
	ignoreSet := make(models.StringSet, len(cfg.IgnoreDirs))
	for _, dir := range cfg.IgnoreDirs {
		resolved, err := utility.ResolvePath(dir)
		if err == nil {
			ignoreSet[resolved] = struct{}{}
		}
	}
	return ignoreSet
}

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

func addTmuxSessionEntries(entries map[string]string, existingSessions map[string]bool, currentSession string) {
	for sessionName := range existingSessions {
		if sessionName == currentSession {
			continue
		}

		displayName := cfg.TmuxSessionPrefix + sessionName
		entries[displayName] = sessionName
	}
}

// processScanDir processes a ScanDir struct, using the struct's depth
// and the existing depth priority logic from the ScanDir.GetDepth method.
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

// deduplicateDisplayNames finds the minimum suffix needed to ensure no duplicates
// using hash-based grouping for efficient conflict resolution.
func deduplicateDisplayNames(allPaths []models.PathInfo) map[string]string {
	if len(allPaths) == 0 {
		return make(map[string]string)
	}

	// Group paths by basename (worst case: all unique basenames)
	groups := make(map[string][]models.PathInfo, len(allPaths))
	for _, info := range allPaths {
		basename := filepath.Base(info.Path)
		groups[basename] = append(groups[basename], info)
	}

	// Build result map with one entry per path
	result := make(map[string]string, len(allPaths))

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

// resolveConflicts finds the minimum suffix depth needed to make all paths unique
func resolveConflicts(paths []models.PathInfo) map[string]string {
	const maxDepth = 10 // Reasonable limit to prevent infinite loops

	for depth := 1; depth <= maxDepth; depth++ {
		// Map each path suffix to its full path info for conflict detection
		suffixes := make(map[string]models.PathInfo, len(paths))
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
			// All unique at this depth - build the result map
			result := make(map[string]string, len(suffixes))
			for suffix, info := range suffixes {
				displayName := normalizePathForDisplay(suffix)
				displayName = applyPrefix(info.Prefix, displayName)
				result[info.Path] = displayName
			}
			return result
		}
	}

	// Fallback: use full path if conflict cant be resolved
	result := make(map[string]string, len(paths))
	for _, info := range paths {
		displayName := normalizePathForDisplay(info.Path)
		displayName = applyPrefix(info.Prefix, displayName)
		result[info.Path] = displayName
	}
	return result
}

// getPathSuffix returns the last N components of a path
func getPathSuffix(path string, depth int) string {
	components := strings.Split(filepath.Clean(path), string(filepath.Separator))

	// Remove empty components (can happen with leading/trailing separators)
	cleanComponents := make([]string, 0, len(components))
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

// shouldSkipEntry determines if an entry should be skipped based on current session
// and existing tmux sessions to avoid duplicates.
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

func applyPrefix(prefix, name string) string {
	if prefix != "" {
		return prefix + "/" + name
	}
	return name
}

// normalizeSessionName replaces leading dots with underscores to match tmux's session naming behavior
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
