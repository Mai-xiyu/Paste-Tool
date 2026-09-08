package platform

import (
	"os"
	"runtime"
	"strings"
)

func IsWayland() bool {
	return runtime.GOOS == "linux" && waylandEnvironment(os.Getenv)
}

func waylandEnvironment(getenv func(string) string) bool {
	return strings.EqualFold(strings.TrimSpace(getenv("XDG_SESSION_TYPE")), "wayland") || getenv("WAYLAND_DISPLAY") != ""
}

func keysymForRune(r rune) uint64 {
	switch r {
	case '\b':
		return 0xff08
	case '\t':
		return 0xff09
	case '\n':
		return 0xff0d
	case 0x1b:
		return 0xff1b
	}
	if r >= 0x20 && r <= 0xff {
		return uint64(r)
	}
	return 0x01000000 | uint64(r)
}
