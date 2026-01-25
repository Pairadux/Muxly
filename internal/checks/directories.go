package checks

import (
	"fmt"
	"os"

	"github.com/Pairadux/muxly/internal/models"
	"github.com/Pairadux/muxly/internal/utility"
)

// ValidateDirectories checks that configured directories exist and are accessible.
func ValidateDirectories(cfg *models.Config) []CheckResult {
	var results []CheckResult

	for _, sd := range cfg.ScanDirs {
		results = append(results, checkDirectory(sd.Path, "scan_dirs"))
	}

	for _, path := range cfg.EntryDirs {
		results = append(results, checkDirectory(path, "entry_dirs"))
	}

	return results
}

func checkDirectory(path, source string) CheckResult {
	resolved, err := utility.ResolvePath(path)
	if err != nil {
		return CheckResult{
			Name:    "directory",
			Status:  StatusError,
			Message: fmt.Sprintf("Invalid path in %s: %s", source, path),
			Hint:    err.Error(),
		}
	}

	info, err := os.Stat(resolved)
	if os.IsNotExist(err) {
		return CheckResult{
			Name:    "directory",
			Status:  StatusError,
			Message: fmt.Sprintf("%s does not exist", path),
			Hint:    fmt.Sprintf("Create the directory or remove from %s", source),
		}
	}
	if err != nil {
		return CheckResult{
			Name:    "directory",
			Status:  StatusError,
			Message: fmt.Sprintf("Cannot access %s", path),
			Hint:    err.Error(),
		}
	}
	if !info.IsDir() {
		return CheckResult{
			Name:    "directory",
			Status:  StatusError,
			Message: fmt.Sprintf("%s is not a directory", path),
			Hint:    fmt.Sprintf("Remove from %s or use a directory path", source),
		}
	}

	f, err := os.Open(resolved)
	if err != nil {
		return CheckResult{
			Name:    "directory",
			Status:  StatusError,
			Message: fmt.Sprintf("%s is not readable", path),
			Hint:    "Check directory permissions",
		}
	}
	f.Close()

	return CheckResult{
		Name:    "directory",
		Status:  StatusOK,
		Message: path,
	}
}
