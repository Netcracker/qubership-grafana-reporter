package utils

import (
	"testing"
)

func TestIsSafeFileName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"empty string", "", false},
		{"contains slash", "file/name", false},
		{"contains backslash", "file\\name", false},
		{"contains dot dot", "file..name", false},
		{"safe name", "filename.txt", true},
		{"safe name with dash", "file-name.txt", true},
		{"safe name with underscore", "file_name.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSafeFileName(tt.input)
			if result != tt.expected {
				t.Errorf("IsSafeFileName(%q) = %v; want %v", tt.input, result, tt.expected)
			}
		})
	}
}
