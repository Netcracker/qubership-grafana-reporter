package handle

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterEndpoints(t *testing.T) {
	addr := "http://localhost:3000"
	credentialsFile := ""
	templates := map[string][]byte{
		"template1": []byte("content1"),
	}
	defaultTemplate := "template1"
	defaultFrom := "now-1h"
	defaultTo := "now"
	renderCollapsed := false
	tlsConfig := &tls.Config{}

	handler := RegisterEndpoints(addr, credentialsFile, templates, defaultTemplate, defaultFrom, defaultTo, renderCollapsed, tlsConfig)

	if handler == nil {
		t.Error("RegisterEndpoints returned nil handler")
	}

	// Test that the handler can handle requests (basic smoke test)
	req := httptest.NewRequest("GET", "/api/v1/templates", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
