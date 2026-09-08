package platform

import (
	"testing"
	"unsafe"
)

func TestWindowsInputABI(t *testing.T) {
	wantSize, wantOffset, wantExtra := uintptr(28), uintptr(4), uintptr(12)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		wantSize, wantOffset, wantExtra = 40, 8, 16
	}
	if got := unsafe.Sizeof(input{}); got != wantSize {
		t.Fatalf("sizeof(INPUT) = %d, Win32 requires %d", got, wantSize)
	}
	if got := unsafe.Offsetof(input{}.Ki); got != wantOffset {
		t.Fatalf("KEYBDINPUT offset = %d, want %d", got, wantOffset)
	}
	if got := unsafe.Offsetof(keyboardInput{}.DwExtraInfo); got != wantExtra {
		t.Fatalf("dwExtraInfo offset = %d, want %d", got, wantExtra)
	}
}

func TestWindowsRuneEvents(t *testing.T) {
	for _, tc := range []struct {
		r     rune
		scans []uint16
		vk    uint16
	}{
		{'a', []uint16{0x61}, 0},
		{'\u4e2d', []uint16{0x4e2d}, 0},
		{'\U0001f600', []uint16{0xd83d, 0xde00}, 0},
		{'\n', []uint16{0}, 0x0d},
		{'\t', []uint16{0}, 0x09},
	} {
		events := inputsForRune(tc.r)
		if len(events) != len(tc.scans)*2 {
			t.Fatalf("%U: wrong event count", tc.r)
		}
		for i, scan := range tc.scans {
			down, up := events[i*2], events[i*2+1]
			flags := uint32(keyEventUnicode)
			if tc.vk != 0 {
				flags = 0
			}
			if down.Type != inputKeyboard || down.Ki.WScan != scan || down.Ki.WVk != tc.vk || down.Ki.DwFlags != flags {
				t.Fatalf("%U: invalid keydown %+v", tc.r, down)
			}
			if up.Ki.WScan != scan || up.Ki.WVk != tc.vk || up.Ki.DwFlags != flags|keyEventKeyUp {
				t.Fatalf("%U: invalid keyup %+v", tc.r, up)
			}
		}
	}
}
