package styles

import (
	"testing"
)

func TestNewTheme_Mocha(t *testing.T) {
	theme := NewTheme("mocha")
	if theme.CheckboxOn == "" {
		t.Error("CheckboxOn should not be empty")
	}
	if theme.CheckboxOff == "" {
		t.Error("CheckboxOff should not be empty")
	}
}

func TestNewTheme_Latte(t *testing.T) {
	theme := NewTheme("latte")
	if theme.CheckboxOn == "" {
		t.Error("CheckboxOn should not be empty")
	}
}

func TestNewTheme_DefaultsToMocha(t *testing.T) {
	theme := NewTheme("unknown")
	if theme.CheckboxOn == "" {
		t.Error("should default to mocha theme")
	}
}

func TestNewThemeWithAccent(t *testing.T) {
	theme := NewThemeWithAccent("mocha", "pink")
	if theme.CheckboxOn == "" {
		t.Error("CheckboxOn should not be empty")
	}
}

func TestNewThemeWithAccent_NoAccent(t *testing.T) {
	theme := NewThemeWithAccent("mocha", "")
	if theme.CheckboxOn == "" {
		t.Error("should work without accent")
	}
}

func TestNewThemeWithAccent_InvalidAccent(t *testing.T) {
	theme := NewThemeWithAccent("mocha", "nonexistent-color")
	if theme.CheckboxOn == "" {
		t.Error("should fall back gracefully")
	}
}

func TestNewThemeWithAccent_Latte(t *testing.T) {
	theme := NewThemeWithAccent("latte", "blue")
	if theme.CheckboxOn == "" {
		t.Error("CheckboxOn should not be empty")
	}
}

func TestAccentHex_ValidMocha(t *testing.T) {
	hex := accentHex("mocha", "mauve")
	if hex != Mocha.Mauve {
		t.Errorf("hex = %q, want %q", hex, Mocha.Mauve)
	}
}

func TestAccentHex_ValidLatte(t *testing.T) {
	hex := accentHex("latte", "red")
	if hex != Latte.Red {
		t.Errorf("hex = %q, want %q", hex, Latte.Red)
	}
}

func TestAccentHex_InvalidPalette(t *testing.T) {
	hex := accentHex("invalid", "mauve")
	if hex != "" {
		t.Errorf("should be empty for invalid palette, got %q", hex)
	}
}

func TestAccentHex_InvalidColor(t *testing.T) {
	hex := accentHex("mocha", "nonexistent")
	if hex != "" {
		t.Errorf("should be empty for invalid color, got %q", hex)
	}
}

func TestAccentColors(t *testing.T) {
	if len(AccentColors) == 0 {
		t.Error("AccentColors should not be empty")
	}
	for _, color := range AccentColors {
		hex := accentHex("mocha", color)
		if hex == "" {
			t.Errorf("accent %q not found in mocha palette", color)
		}
	}
}

func TestMochaPalette(t *testing.T) {
	if Mocha.Base == "" || Mocha.Text == "" || Mocha.Mauve == "" {
		t.Error("Mocha palette has empty values")
	}
}

func TestLattePalette(t *testing.T) {
	if Latte.Base == "" || Latte.Text == "" || Latte.Mauve == "" {
		t.Error("Latte palette has empty values")
	}
}
