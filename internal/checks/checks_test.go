package checks

import "testing"

func TestCountByStatus(t *testing.T) {
	tests := []struct {
		name             string
		results          []CheckResult
		expectedErrors   int
		expectedWarnings int
	}{
		{
			name:             "empty results",
			results:          []CheckResult{},
			expectedErrors:   0,
			expectedWarnings: 0,
		},
		{
			name: "all OK",
			results: []CheckResult{
				{Name: "check1", Status: StatusOK},
				{Name: "check2", Status: StatusOK},
			},
			expectedErrors:   0,
			expectedWarnings: 0,
		},
		{
			name: "only errors",
			results: []CheckResult{
				{Name: "check1", Status: StatusError},
				{Name: "check2", Status: StatusError},
				{Name: "check3", Status: StatusError},
			},
			expectedErrors:   3,
			expectedWarnings: 0,
		},
		{
			name: "only warnings",
			results: []CheckResult{
				{Name: "check1", Status: StatusWarning},
				{Name: "check2", Status: StatusWarning},
			},
			expectedErrors:   0,
			expectedWarnings: 2,
		},
		{
			name: "mixed results",
			results: []CheckResult{
				{Name: "check1", Status: StatusOK},
				{Name: "check2", Status: StatusError},
				{Name: "check3", Status: StatusWarning},
				{Name: "check4", Status: StatusError},
				{Name: "check5", Status: StatusOK},
				{Name: "check6", Status: StatusWarning},
			},
			expectedErrors:   2,
			expectedWarnings: 2,
		},
		{
			name: "single error",
			results: []CheckResult{
				{Name: "check1", Status: StatusError},
			},
			expectedErrors:   1,
			expectedWarnings: 0,
		},
		{
			name: "single warning",
			results: []CheckResult{
				{Name: "check1", Status: StatusWarning},
			},
			expectedErrors:   0,
			expectedWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErrors, gotWarnings := CountByStatus(tt.results)
			if gotErrors != tt.expectedErrors {
				t.Errorf("CountByStatus() errors = %d, want %d", gotErrors, tt.expectedErrors)
			}
			if gotWarnings != tt.expectedWarnings {
				t.Errorf("CountByStatus() warnings = %d, want %d", gotWarnings, tt.expectedWarnings)
			}
		})
	}
}

func TestHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		results  []CheckResult
		expected bool
	}{
		{
			name:     "empty results",
			results:  []CheckResult{},
			expected: false,
		},
		{
			name: "all OK",
			results: []CheckResult{
				{Name: "check1", Status: StatusOK},
				{Name: "check2", Status: StatusOK},
			},
			expected: false,
		},
		{
			name: "only warnings",
			results: []CheckResult{
				{Name: "check1", Status: StatusWarning},
				{Name: "check2", Status: StatusWarning},
			},
			expected: false,
		},
		{
			name: "has one error",
			results: []CheckResult{
				{Name: "check1", Status: StatusOK},
				{Name: "check2", Status: StatusError},
				{Name: "check3", Status: StatusWarning},
			},
			expected: true,
		},
		{
			name: "all errors",
			results: []CheckResult{
				{Name: "check1", Status: StatusError},
				{Name: "check2", Status: StatusError},
			},
			expected: true,
		},
		{
			name: "error at beginning",
			results: []CheckResult{
				{Name: "check1", Status: StatusError},
				{Name: "check2", Status: StatusOK},
			},
			expected: true,
		},
		{
			name: "error at end",
			results: []CheckResult{
				{Name: "check1", Status: StatusOK},
				{Name: "check2", Status: StatusWarning},
				{Name: "check3", Status: StatusError},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HasErrors(tt.results)
			if got != tt.expected {
				t.Errorf("HasErrors() = %v, want %v", got, tt.expected)
			}
		})
	}
}
