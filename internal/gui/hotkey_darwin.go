//go:build darwin && cgo

package gui

import (
	"strings"

	"golang.design/x/hotkey"
)

var darwinLetterKeys = map[byte]hotkey.Key{
	'A': hotkey.KeyA,
	'B': hotkey.KeyB,
	'C': hotkey.KeyC,
	'D': hotkey.KeyD,
	'E': hotkey.KeyE,
	'F': hotkey.KeyF,
	'G': hotkey.KeyG,
	'H': hotkey.KeyH,
	'I': hotkey.KeyI,
	'J': hotkey.KeyJ,
	'K': hotkey.KeyK,
	'L': hotkey.KeyL,
	'M': hotkey.KeyM,
	'N': hotkey.KeyN,
	'O': hotkey.KeyO,
	'P': hotkey.KeyP,
	'Q': hotkey.KeyQ,
	'R': hotkey.KeyR,
	'S': hotkey.KeyS,
	'T': hotkey.KeyT,
	'U': hotkey.KeyU,
	'V': hotkey.KeyV,
	'W': hotkey.KeyW,
	'X': hotkey.KeyX,
	'Y': hotkey.KeyY,
	'Z': hotkey.KeyZ,
}

var darwinDigitKeys = map[byte]hotkey.Key{
	'0': hotkey.Key0,
	'1': hotkey.Key1,
	'2': hotkey.Key2,
	'3': hotkey.Key3,
	'4': hotkey.Key4,
	'5': hotkey.Key5,
	'6': hotkey.Key6,
	'7': hotkey.Key7,
	'8': hotkey.Key8,
	'9': hotkey.Key9,
}

var darwinFunctionKeys = []hotkey.Key{
	hotkey.KeyF1,
	hotkey.KeyF2,
	hotkey.KeyF3,
	hotkey.KeyF4,
	hotkey.KeyF5,
	hotkey.KeyF6,
	hotkey.KeyF7,
	hotkey.KeyF8,
	hotkey.KeyF9,
	hotkey.KeyF10,
	hotkey.KeyF11,
	hotkey.KeyF12,
}

func hotkeyModifierByName(value string) (hotkey.Modifier, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ctrl", "control":
		return hotkey.ModCtrl, true
	case "alt":
		return hotkey.ModOption, true
	case "shift":
		return hotkey.ModShift, true
	case "win", "super", "cmd", "meta":
		return hotkey.ModCmd, true
	default:
		return 0, false
	}
}

func hotkeyLetterKey(ch byte) hotkey.Key {
	return darwinLetterKeys[ch]
}

func hotkeyDigitKey(ch byte) hotkey.Key {
	return darwinDigitKeys[ch]
}

func hotkeyFunctionKey(n int) hotkey.Key {
	return darwinFunctionKeys[n-1]
}
