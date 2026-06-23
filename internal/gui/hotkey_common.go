package gui

import (
	"fmt"
	"strconv"
	"strings"

	"golang.design/x/hotkey"
)

func hotkeyKeyByName(value string) (hotkey.Key, error) {
	key := strings.ToUpper(strings.TrimSpace(value))
	if len(key) == 1 {
		ch := key[0]
		if ch >= 'A' && ch <= 'Z' {
			return hotkeyLetterKey(ch), nil
		}
		if ch >= '0' && ch <= '9' {
			return hotkeyDigitKey(ch), nil
		}
	}
	if strings.HasPrefix(key, "F") {
		n, err := strconv.Atoi(strings.TrimPrefix(key, "F"))
		if err == nil && n >= 1 && n <= 12 {
			return hotkeyFunctionKey(n), nil
		}
	}
	return 0, fmt.Errorf("unsupported hotkey key %q", value)
}
