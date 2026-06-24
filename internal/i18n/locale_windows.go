//go:build windows

package i18n

import (
	"syscall"
	"unsafe"
)

var (
	kernel32                 = syscall.NewLazyDLL("kernel32.dll")
	procGetUserDefaultLocale = kernel32.NewProc("GetUserDefaultLocaleName")
)

func systemLocale() Locale {
	buf := make([]uint16, 85)
	ret, _, _ := procGetUserDefaultLocale.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
	)
	if ret == 0 {
		return Auto
	}
	return Normalize(syscall.UTF16ToString(buf))
}
