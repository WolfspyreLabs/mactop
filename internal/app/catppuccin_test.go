package app

import (
	"testing"
)

// TestGetCatppuccinHex tests the GetCatppuccinHex function
// which returns hex color values for Catppuccin theme colors
func TestGetCatppuccinHex(t *testing.T) {
	tests := []struct {
		name      string
		theme     string
		colorName string
		want      string
	}{
		{"Mocha base", "mocha", "Base", "#1e1e2e"},
		{"Mocha text", "mocha", "Text", "#cdd6f4"},
		{"Frappe base", "frappe", "Base", "#303446"},
		{"Latte base", "latte", "Base", "#eff1f5"},
		{"Unknown theme", "unknown", "Base", "#ffffff"},
		{"Unknown color", "mocha", "unknown_color", "#ffffff"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCatppuccinHex(tt.theme, tt.colorName)
			if got != tt.want {
				t.Errorf("GetCatppuccinHex(%q, %q) = %q, want %q", tt.theme, tt.colorName, got, tt.want)
			}
		})
	}
}

// TestGetCatppuccinPalette tests the GetCatppuccinPalette function
// which returns a CatppuccinPalette for a given theme name
func TestGetCatppuccinPalette(t *testing.T) {
	tests := []struct {
		name  string
		theme string
		want  bool // true if should return non-nil palette
	}{
		{"Valid latte", "latte", true},
		{"Valid frappe", "frappe", true},
		{"Valid macchiato", "macchiato", true},
		{"Valid mocha", "mocha", true},
		{"Invalid theme", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			palette := GetCatppuccinPalette(tt.theme)
			if tt.want && palette == nil {
				t.Errorf("GetCatppuccinPalette(%q) returned nil, expected non-nil", tt.theme)
			}
			if !tt.want && palette != nil {
				t.Errorf("GetCatppuccinPalette(%q) returned non-nil, expected nil", tt.theme)
			}
		})
	}
}
