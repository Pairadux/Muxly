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
}
