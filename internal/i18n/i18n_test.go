package i18n

import (
	"os"
	"testing"
)

func TestInit_EN(t *testing.T) {
	Init("en")
	if Current() != EN {
		t.Errorf("expected EN, got %s", Current())
	}
}

func TestInit_ZH(t *testing.T) {
	Init("zh")
	if Current() != ZH {
		t.Errorf("expected ZH, got %s", Current())
	}
}

func TestInit_AutoDetectsFromLANG(t *testing.T) {
	os.Setenv("LANG", "zh_CN.UTF-8")
	defer os.Unsetenv("LANG")
	os.Unsetenv("LC_ALL")

	Init("auto")
	if Current() != ZH {
		t.Errorf("expected ZH from LANG=zh_CN, got %s", Current())
	}
}

func TestInit_AutoDetectsFromLC_ALL(t *testing.T) {
	os.Unsetenv("LANG")
	os.Setenv("LC_ALL", "zh_TW.UTF-8")
	defer os.Unsetenv("LC_ALL")

	Init("auto")
	if Current() != ZH {
		t.Errorf("expected ZH from LC_ALL=zh_TW, got %s", Current())
	}
}

func TestInit_AutoDefaultsToEN(t *testing.T) {
	os.Unsetenv("LANG")
	os.Unsetenv("LC_ALL")

	Init("auto")
	if Current() != EN {
		t.Errorf("expected EN default, got %s", Current())
	}
}

func TestInit_EmptyStringIsAuto(t *testing.T) {
	os.Unsetenv("LANG")
	os.Unsetenv("LC_ALL")

	Init("")
	if Current() != EN {
		t.Errorf("expected EN default for empty init, got %s", Current())
	}
}

func TestT_ReturnsENTranslation(t *testing.T) {
	Init("en")
	got := T("cmd.list")
	if got != "List installed skills" {
		t.Errorf("T(cmd.list) = %q, want %q", got, "List installed skills")
	}
}

func TestT_ReturnsZHTranslation(t *testing.T) {
	Init("zh")
	got := T("cmd.list")
	if got != "列出已安装的技能" {
		t.Errorf("T(cmd.list) = %q, want %q", got, "列出已安装的技能")
	}
}

func TestT_UnknownKeyReturnsKey(t *testing.T) {
	Init("en")
	got := T("nonexistent.key")
	if got != "nonexistent.key" {
		t.Errorf("T(nonexistent) = %q, want %q", got, "nonexistent.key")
	}
}

func TestT_FallsBackToEN(t *testing.T) {
	// Set to a language that doesn't exist in the map
	currentLang = Lang("fr")
	got := T("cmd.list")
	expected := "List installed skills"
	if got != expected {
		t.Errorf("T with unknown lang = %q, want %q (EN fallback)", got, expected)
	}
}

func TestTf_FormatsString(t *testing.T) {
	Init("en")
	got := Tf("install.success", "myskill", "claude")
	if got == "" {
		t.Error("Tf returned empty string")
	}
}

func TestDetectLang_NoEnvVars(t *testing.T) {
	os.Unsetenv("LANG")
	os.Unsetenv("LC_ALL")
	got := detectLang()
	if got != EN {
		t.Errorf("detectLang() = %s, want EN", got)
	}
}

func TestDetectLang_ChineseLANG(t *testing.T) {
	os.Setenv("LANG", "zh_CN.UTF-8")
	defer os.Unsetenv("LANG")
	os.Unsetenv("LC_ALL")

	got := detectLang()
	if got != ZH {
		t.Errorf("detectLang() = %s, want ZH", got)
	}
}

func TestDetectLang_ChineseLC_ALL(t *testing.T) {
	os.Unsetenv("LANG")
	os.Setenv("LC_ALL", "zh_TW.UTF-8")
	defer os.Unsetenv("LC_ALL")

	got := detectLang()
	if got != ZH {
		t.Errorf("detectLang() = %s, want ZH", got)
	}
}

func TestCurrent(t *testing.T) {
	Init("en")
	if Current() != EN {
		t.Errorf("Current() = %s, want EN", Current())
	}
	Init("zh")
	if Current() != ZH {
		t.Errorf("Current() = %s, want ZH", Current())
	}
}

func TestAllMessagesHaveENTranslation(t *testing.T) {
	for key, langs := range messages {
		if _, ok := langs[EN]; !ok {
			t.Errorf("message %q has no EN translation", key)
		}
	}
}

func TestAllMessagesHaveZHTranslation(t *testing.T) {
	for key, langs := range messages {
		if _, ok := langs[ZH]; !ok {
			t.Errorf("message %q has no ZH translation", key)
		}
	}
}
