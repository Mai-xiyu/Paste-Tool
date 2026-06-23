//go:build darwin && cgo

package platform

/*
#cgo LDFLAGS: -framework ApplicationServices -framework AudioToolbox
#include <ApplicationServices/ApplicationServices.h>
#include <AudioToolbox/AudioToolbox.h>

static int paste_tool_ax_trusted(void) {
	return AXIsProcessTrusted();
}

static int paste_tool_post_utf16(const UniChar *chars, UniCharCount length) {
	CGEventRef down = CGEventCreateKeyboardEvent(NULL, 0, true);
	if (down == NULL) {
		return 1;
	}
	CGEventKeyboardSetUnicodeString(down, length, chars);
	CGEventPost(kCGHIDEventTap, down);
	CFRelease(down);

	CGEventRef up = CGEventCreateKeyboardEvent(NULL, 0, false);
	if (up == NULL) {
		return 2;
	}
	CGEventKeyboardSetUnicodeString(up, length, chars);
	CGEventPost(kCGHIDEventTap, up);
	CFRelease(up);
	return 0;
}

static void paste_tool_beep_start(void) {
	AudioServicesPlaySystemSound(1057);
}

static void paste_tool_beep_error(void) {
	AudioServicesPlaySystemSound(kSystemSoundID_UserPreferredAlert);
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"unicode/utf16"
	"unsafe"
)

type darwinDriver struct{}

func NewDriver() Driver {
	return darwinDriver{}
}

func (darwinDriver) Name() string {
	return "macos-coregraphics"
}

func (darwinDriver) SendRune(ctx context.Context, r rune) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if C.paste_tool_ax_trusted() == 0 {
		return errors.New("macOS Accessibility/Input Monitoring permission is required")
	}
	units := utf16.Encode([]rune{r})
	if len(units) == 0 {
		return nil
	}
	rc := C.paste_tool_post_utf16((*C.UniChar)(unsafe.Pointer(&units[0])), C.UniCharCount(len(units)))
	if rc != 0 {
		return fmt.Errorf("CGEventPost failed with code %d", int(rc))
	}
	return nil
}

func (darwinDriver) NotifyStart() error {
	C.paste_tool_beep_start()
	return nil
}

func (darwinDriver) NotifyError() error {
	C.paste_tool_beep_error()
	return nil
}

func (darwinDriver) Check(context.Context) []Check {
	status := StatusOK
	detail := "Accessibility/Input Monitoring permission is granted"
	if C.paste_tool_ax_trusted() == 0 {
		status = StatusWarning
		detail = "grant Accessibility and Input Monitoring permission before paste injection"
	}
	return []Check{
		RuntimeCheck(),
		{Name: "input", Status: status, Detail: detail},
	}
}
