//go:build !windows && !linux && !darwin

package gui

import (
	"strings"

	"golang.design/x/hotkey"
)

func hotkeyModifierByName(value string) (hotkey.Modifier, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ctrl", "control", "alt", "shift", "win", "super", "cmd", "meta":
		return 0, true
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
	return hotkey.Key(n)
}
