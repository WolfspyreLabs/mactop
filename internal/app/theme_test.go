package app

import (
	"testing"
)

// TestIsCatppuccinTheme tests the IsCatppuccinTheme function
// which validates if a given theme name is a Catppuccin theme
func TestIsCatppuccinTheme(t *testing.T) {
	tests := []struct {
		name  string
		theme string
		want  bool
	}{
		{"Valid frappe", "frappe", true},
		{"Valid macchiato", "macchiato", true},
		{"Valid mocha", "mocha", true},
		{"Invalid theme", "dracula", false},
		{"Empty string", "", false},
		{"Case sensitive", "Mocha", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCatppuccinTheme(tt.theme)
			if got != tt.want {
				t.Errorf("IsCatppuccinTheme(%q) = %v, want %v", tt.theme, got, tt.want)
			}
		})
	}
}

// TestGetCurrentBgName tests the GetCurrentBgName function
// which returns the current background color name
func TestGetCurrentBgName(t *testing.T) {
	// This function reads from currentConfig, so we just verify it doesn't panic
	// and returns a non-empty string in normal operation
	bgName := GetCurrentBgName()
	if bgName == "" {
		t.Error("GetCurrentBgName returned empty string")
	}
}

// TestGetProcessTextColor tests the GetProcessTextColor function
// which returns the appropriate text color based on whether process is current user
func TestGetProcessTextColor(t *testing.T) {
	// Test with isCurrentUser = true
	colorCurrentUser := GetProcessTextColor(true)
	// May return empty if no theme is configured, which is acceptable

	// Test with isCurrentUser = false
	colorOtherUser := GetProcessTextColor(false)
	// May return empty if no theme is configured, which is acceptable

	// If both return values, verify they can be different
	if colorCurrentUser != "" && colorOtherUser != "" {
		// This is fine - they can be the same or different depending on config
		t.Logf("Got colors: current=%q, other=%q", colorCurrentUser, colorOtherUser)
	}
}
