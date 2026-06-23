//go:build linux && cgo

package gui

import (
	"strings"

	"golang.design/x/hotkey"
)

func hotkeyModifierByName(value string) (hotkey.Modifier, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ctrl", "control":
		return hotkey.ModCtrl, true
	case "alt":
		return hotkey.Mod1, true
	case "shift":
		return hotkey.ModShift, true
	case "win", "super", "cmd", "meta":
		return hotkey.Mod4, true
	default:
		return 0, false
	}
}

func hotkeyLetterKey(ch byte) hotkey.Key {
	return hotkey.Key(ch)
}

func hotkeyDigitKey(ch byte) hotkey.Key {
	return hotkey.Key(ch)
}

func hotkeyFunctionKey(n int) hotkey.Key {
	return hotkey.Key(uint32(hotkey.KeyF1) + uint32(n-1))
}
