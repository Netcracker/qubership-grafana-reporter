package report

import (
	"net/http/httptest"
	"os"
	"testing"
)

func TestGenerateUniqueRequestID(t *testing.T) {
	tests := []struct {
		uid         string
		from        string
		to          string
		isCollapsed bool
		expected    string
	}{
		{"dashboard1", "now-1h", "now", false, "dashboard1_report_now-1h-now_expanded"},
		{"dashboard1", "now-1h", "now", true, "dashboard1_report_now-1h-now"},
	}

	for _, tt := range tests {
		result := generateUniqueRequestID(tt.uid, tt.from, tt.to, tt.isCollapsed)
		if result != tt.expected {
			t.Errorf("generateUniqueRequestID(%q, %q, %q, %v) = %q; want %q", tt.uid, tt.from, tt.to, tt.isCollapsed, result, tt.expected)
		}
	}
}

func TestGetPanelsDirPath(t *testing.T) {
	requestID := "test123"
	expected := os.TempDir() + "/test123"
	result := getPanelsDirPath(requestID)
	if result != expected {
		t.Errorf("getPanelsDirPath(%q) = %q; want %q", requestID, result, expected)
	}
}

func TestGetParameterFromRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?param=value&other=other", nil)

	tests := []struct {
		name         string
		param        string
		defaultValue string
		expected     string
	}{
		{"existing param", "param", "default", "value"},
		{"non-existing param", "missing", "default", "default"},
		{"empty name", "", "default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getParameterFromRequest(req, tt.param, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getParameterFromRequest(%q, %q) = %q; want %q", tt.param, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetBoolParameterFromRequest(t *testing.T) {
	req := httptest.NewRequest("GET", "/test?bool=true&str=value", nil)

	tests := []struct {
		name         string
		param        string
		defaultValue bool
		expected     bool
	}{
		{"true param", "bool", false, true},
		{"non-existing param", "missing", true, true},
		{"invalid bool", "str", false, false},
		{"empty name", "", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBoolParameterFromRequest(req, tt.param, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("getBoolParameterFromRequest(%q, %v) = %v; want %v", tt.param, tt.defaultValue, result, tt.expected)
			}
		})
	}
}

func TestGetMaxConcurrentRequests(t *testing.T) {
	// Test default
	os.Unsetenv("MAX_CONCURRENT_RENDER_REQUESTS")
	result := getMaxConcurrentRequests()
	if result != 4 {
		t.Errorf("getMaxConcurrentRequests() = %d; want 4", result)
	}

	// Test with env
	os.Setenv("MAX_CONCURRENT_RENDER_REQUESTS", "10")
	result = getMaxConcurrentRequests()
	if result != 10 {
		t.Errorf("getMaxConcurrentRequests() = %d; want 10", result)
	}

	// Cleanup
	os.Unsetenv("MAX_CONCURRENT_RENDER_REQUESTS")
}

func TestCredentialsGetAuthHeader(t *testing.T) {
	tests := []struct {
		name     string
		creds    Credentials
		expected string
		hasError bool
	}{
		{"token", Credentials{Token: "mytoken"}, "Bearer mytoken", false},
		{"user pass", Credentials{User: "user", Password: "pass"}, "Basic dXNlcjpwYXNz", false},
		{"no creds", Credentials{}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tt.creds.getAuthHeader()
			if tt.hasError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("getAuthHeader() = %q; want %q", result, tt.expected)
				}
			}
		})
	}
}
