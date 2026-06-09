package dashboard

import (
	"testing"
)

func TestPanelIsTheFirst(t *testing.T) {
	tests := []struct {
		name     string
		x        int
		expected bool
	}{
		{"first position", 0, true},
		{"not first", 1, false},
		{"not first", 12, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Panel{GridPos: GridPos{X: tt.x}}
			result := p.IsTheFirst()
			if result != tt.expected {
				t.Errorf("IsTheFirst() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestPanelIsTheLast(t *testing.T) {
	tests := []struct {
		name     string
		x, w     int
		expected bool
	}{
		{"last position", 18, 6, true},
		{"not last", 0, 6, false},
		{"not last", 12, 6, false},
		{"overlaps", 20, 6, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Panel{GridPos: GridPos{X: tt.x, W: tt.w}}
			result := p.IsTheLast()
			if result != tt.expected {
				t.Errorf("IsTheLast() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestPanelGetPxWidth(t *testing.T) {
	p := &Panel{GridPos: GridPos{W: 12}}
	screenWidth := 1920
	expected := 960
	result := p.GetPxWidth(screenWidth)
	if result != expected {
		t.Errorf("GetPxWidth(%d) = %d; want %d", screenWidth, result, expected)
	}
}

func TestPanelGetPxHeight(t *testing.T) {
	p := &Panel{GridPos: GridPos{H: 8}}
	screenWidth := 1920
	expected := 640
	result := p.GetPxHeight(screenWidth)
	if result != expected {
		t.Errorf("GetPxHeight(%d) = %d; want %d", screenWidth, result, expected)
	}
}

func TestPanelGetRelativeWidth(t *testing.T) {
	p := &Panel{GridPos: GridPos{W: 12}}
	screenWidth := 1920
	expected := 0.5 - 0.005
	result := p.GetRelativeWidth(screenWidth)
	if result != expected {
		t.Errorf("GetRelativeWidth(%d) = %f; want %f", screenWidth, result, expected)
	}
}

func TestPanelIsAddedToPreviousRow(t *testing.T) {
	row := Row{
		Panels: []Panel{
			{GridPos: GridPos{W: 12}},
		},
	}

	tests := []struct {
		name     string
		w        int
		expected bool
	}{
		{"fits", 12, true},
		{"does not fit", 13, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Panel{GridPos: GridPos{W: tt.w}}
			result := p.IsAddedToPreviousRow(row)
			if result != tt.expected {
				t.Errorf("IsAddedToPreviousRow() = %v; want %v", result, tt.expected)
			}
		})
	}
}

func TestRoundFloat(t *testing.T) {
	tests := []struct {
		val       float64
		precision uint
		expected  float64
	}{
		{1.23456, 2, 1.23},
		{1.235, 2, 1.24},
		{1.234, 3, 1.234},
	}

	for _, tt := range tests {
		result := roundFloat(tt.val, tt.precision)
		if result != tt.expected {
			t.Errorf("roundFloat(%f, %d) = %f; want %f", tt.val, tt.precision, result, tt.expected)
		}
	}
}

func TestEntityGetStructuredDashboard(t *testing.T) {
	// Test basic dashboard
	entity := &Entity{
		Dashboard: Dashboard{
			Title: "Test Dashboard",
			UID:   "test-uid",
			Panels: []Panel{
				{ID: 1, Type: "graph", GridPos: GridPos{H: 8, W: 12, X: 0, Y: 0}},
			},
		},
		Meta: Meta{Slug: "test-slug"},
	}

	sd, err := entity.GetStructuredDashboard(false)
	if err != nil {
		t.Errorf("GetStructuredDashboard failed: %v", err)
	}
	if sd.Title != "Test Dashboard" {
		t.Errorf("Title = %s; want Test Dashboard", sd.Title)
	}
	if len(sd.Rows) != 1 {
		t.Errorf("Rows count = %d; want 1", len(sd.Rows))
	}
}
