# TODO - 1.0.0 Release Checklist

## Critical Blockers (Must Fix Before 1.0.0)

### 1. Complete `tms create` Command

- [ ] Fix placeholder implementation in `cmd/create.go:98-102`
- [ ] Implement actual session creation logic instead of debug prints
- [ ] Complete `parseWindows()` function to parse window configuration
- [ ] Execute the interactive `CreateForm` properly

### 2. Implement Interactive Forms

- [ ] Complete forms package implementation (`internal/forms/forms.go`)
- [ ] Build proper TUI forms using Huh library
- [ ] Add form validation and error handling
- [ ] Integrate forms with create command workflow

### 3. Fix Config Init Process

- [ ] Remove hardcoded defaults in `cmd/init.go:60`
- [ ] Implement interactive menu for configuration setup
- [ ] Fix `--Defaults` flag hardcoding (`cmd/init.go:117`)
- [ ] Add proper user prompts for scan directories and settings

### 4. Add Test Coverage

- [ ] Create unit tests for core functionality
- [ ] Add integration tests for tmux operations
- [ ] Test configuration parsing and validation
- [ ] Achieve at least 70% test coverage
- [ ] Add CI/CD pipeline for automated testing

## Important for Stability

### 5. Configuration Validation

- [ ] Handle malformed YAML configurations gracefully
- [ ] Validate circular directory references in scan paths
- [ ] Improve error messages for configuration issues
- [ ] Fix config unmarshalling error handling (`root.go:191`)

### 6. Better Error Messages & UX

- [ ] Standardize error message formatting
- [ ] Add graceful handling of interrupted operations
- [ ] Replace basic stdin confirmations with proper UI components
- [ ] Improve user feedback for long-running operations

### 7. Performance Optimizations

- [ ] Address PERF comments throughout codebase
- [ ] Optimize channel buffer sizes for large directory trees
- [ ] Pre-allocate maps and slices where possible
- [ ] Profile and optimize directory scanning performance

## Nice-to-Have for 1.0

### 8. Code Quality Improvements

- [ ] Refactor large complex functions (e.g., `buildDirectoryEntries`)
- [ ] Address REFACTOR comments in codebase
- [ ] Extract hardcoded values to configuration
- [ ] Improve code organization and modularity

### 9. Advanced Features

- [ ] Support for `.tms` files in project directories (`root.go:126`)
- [ ] Add absolute path display option for fzf selector
- [ ] Option to remove current session from selection list
- [ ] Additional configuration options and customization

### 10. Documentation & Polish

- [ ] Update README with complete usage examples
- [ ] Add man page or detailed help documentation
- [ ] Include installation instructions for different platforms
- [ ] Add changelog and version history

## Progress Tracking

- **Critical Blockers**: 0/4 completed
- **Important for Stability**: 0/3 completed
- **Nice-to-Have**: 0/3 completed

**Overall Progress**: 0/10 major areas completed

---

_Last Updated: July 2, 2025_
