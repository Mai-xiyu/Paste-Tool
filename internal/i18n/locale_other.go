//go:build !windows

package i18n

func systemLocale() Locale {
	return Auto
}
