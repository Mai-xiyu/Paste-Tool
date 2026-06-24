package i18n

import "testing"

func TestNormalize(t *testing.T) {
	tests := map[string]Locale{
		"":          Auto,
		"auto":      Auto,
		"zh_CN":     ZhCN,
		"zh-Hans":   ZhCN,
		"en-US":     En,
		"something": Auto,
	}
	for input, want := range tests {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTranslatorFallback(t *testing.T) {
	tr := New("zh-CN")
	if got := tr.T(MenuSettings); got != "设置" {
		t.Fatalf("MenuSettings = %q", got)
	}
	if got := tr.T(Key("missing.key")); got != "missing.key" {
		t.Fatalf("missing key = %q", got)
	}
}
