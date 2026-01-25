package checks

// CheckStatus represents the result status of a check
type CheckStatus int

const (
	StatusOK CheckStatus = iota
	StatusWarning
	StatusError
)

// CheckResult represents the outcome of a single check
type CheckResult struct {
	Name    string
	Status  CheckStatus
	Message string
	Detail  string
	Hint    string
}

// CountByStatus returns the number of errors and warnings in results
func CountByStatus(results []CheckResult) (errors, warnings int) {
	for _, r := range results {
		switch r.Status {
		case StatusError:
			errors++
		case StatusWarning:
			warnings++
		}
	}
	return
}

// HasErrors returns true if any result has StatusError
func HasErrors(results []CheckResult) bool {
	for _, r := range results {
		if r.Status == StatusError {
			return true
		}
	}
	return false
}
