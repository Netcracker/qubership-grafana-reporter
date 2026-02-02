package docs

import (
	"testing"
)

func TestSwaggerInfo(t *testing.T) {
	if SwaggerInfo == nil {
		t.Error("SwaggerInfo is nil")
	}
	if SwaggerInfo.Title != "Grafana Reporter REST API" {
		t.Errorf("Expected title 'Grafana Reporter REST API', got %s", SwaggerInfo.Title)
	}
	if SwaggerInfo.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", SwaggerInfo.Version)
	}
}
