package checks

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestCheckEditor(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "myeditor"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	tests := []struct {
		name       string
		config     string
		editorEnv  string
		visualEnv  string
		wantStatus CheckStatus
		wantDetail string
	}{
		{name: "nothing set", wantStatus: StatusWarning},
		{name: "blank config", config: " ", wantStatus: StatusWarning},
		{name: "blank everywhere", config: "\t", editorEnv: " ", visualEnv: "  ", wantStatus: StatusWarning},
		{name: "config wins", config: "myeditor", editorEnv: "other", wantStatus: StatusOK, wantDetail: "(myeditor)"},
		{name: "blank config falls back to EDITOR", config: " ", editorEnv: "myeditor", wantStatus: StatusOK, wantDetail: "(myeditor)"},
		{name: "blank EDITOR falls back to VISUAL", editorEnv: " ", visualEnv: "myeditor", wantStatus: StatusOK, wantDetail: "(myeditor)"},
		{name: "arguments are allowed", config: "myeditor --wait", wantStatus: StatusOK, wantDetail: "(myeditor --wait)"},
		{name: "editor not installed", config: "nope", wantStatus: StatusWarning},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("EDITOR", tt.editorEnv)
			t.Setenv("VISUAL", tt.visualEnv)

			got := CheckEditor(tt.config)

			if got.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v (%+v)", got.Status, tt.wantStatus, got)
			}
			if got.Detail != tt.wantDetail {
				t.Errorf("detail = %q, want %q", got.Detail, tt.wantDetail)
			}
		})
	}
}
