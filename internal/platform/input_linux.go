//go:build linux && cgo

package platform

/*
#cgo LDFLAGS: -lX11 -lXtst
#include <X11/Xlib.h>
#include <X11/Xutil.h>
#include <X11/extensions/XTest.h>
#include <stdlib.h>

static int paste_tool_xtest_send_keysym(unsigned long keysym) {
	Display *display = XOpenDisplay(NULL);
	if (display == NULL) {
		return 1;
	}

	int event_base = 0;
	int error_base = 0;
	int major = 0;
	int minor = 0;
	if (!XTestQueryExtension(display, &event_base, &error_base, &major, &minor)) {
		XCloseDisplay(display);
		return 2;
	}

	KeyCode keycode = XKeysymToKeycode(display, (KeySym)keysym);
	KeySym *old_mapping = NULL;
	int keysyms_per_keycode = 0;
	int changed_mapping = 0;

	if (keycode == 0) {
		int min_keycode = 0;
		int max_keycode = 0;
		XDisplayKeycodes(display, &min_keycode, &max_keycode);
		keycode = (KeyCode)max_keycode;
		old_mapping = XGetKeyboardMapping(display, keycode, 1, &keysyms_per_keycode);
		KeySym new_mapping[1];
		new_mapping[0] = (KeySym)keysym;
		XChangeKeyboardMapping(display, keycode, 1, new_mapping, 1);
		XSync(display, False);
		changed_mapping = 1;
	}

	XTestFakeKeyEvent(display, keycode, True, CurrentTime);
	XTestFakeKeyEvent(display, keycode, False, CurrentTime);
	XFlush(display);

	if (changed_mapping) {
		if (old_mapping != NULL && keysyms_per_keycode > 0) {
			XChangeKeyboardMapping(display, keycode, keysyms_per_keycode, old_mapping, 1);
			XFree(old_mapping);
		}
		XSync(display, False);
	}

	XCloseDisplay(display);
	return 0;
}
*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"os"
)

const (
	xkBackSpace = 0xff08
	xkTab       = 0xff09
	xkReturn    = 0xff0d
	xkEscape    = 0xff1b
)

type linuxDriver struct{}

func NewDriver() Driver {
	return linuxDriver{}
}

func (linuxDriver) Name() string {
	return "linux-x11-xtest"
}

func (linuxDriver) SendRune(ctx context.Context, r rune) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := checkLinuxInputEnvironment(); err != nil {
		return err
	}
	code := C.ulong(keysymForRune(r))
	rc := C.paste_tool_xtest_send_keysym(code)
	switch rc {
	case 0:
		return nil
	case 1:
		return errors.New("cannot open X11 display")
	case 2:
		return errors.New("XTest extension is unavailable")
	default:
		return fmt.Errorf("XTest failed with code %d", int(rc))
	}
}

func (linuxDriver) NotifyStart() error { return nil }

func (linuxDriver) NotifyError() error { return nil }

func (linuxDriver) Check(context.Context) []Check {
	status := StatusOK
	detail := "X11 DISPLAY is set and XTest can be used"
	if err := checkLinuxInputEnvironment(); err != nil {
		status = StatusUnsupported
		detail = err.Error()
	}
	return []Check{
		RuntimeCheck(),
		{Name: "input", Status: status, Detail: detail},
	}
}

func checkLinuxInputEnvironment() error {
	display := os.Getenv("DISPLAY")
	wayland := os.Getenv("WAYLAND_DISPLAY")
	if display == "" && wayland != "" {
		return fmt.Errorf("%w: Wayland session detected without X11 DISPLAY; synthetic input is intentionally blocked", ErrUnsupported)
	}
	if display == "" {
		return fmt.Errorf("%w: DISPLAY is not set", ErrUnsupported)
	}
	return nil
}

func keysymForRune(r rune) uint64 {
	switch r {
	case '\b':
		return xkBackSpace
	case '\t':
		return xkTab
	case '\n':
		return xkReturn
	case 0x1b:
		return xkEscape
	}
	if r >= 0x20 && r <= 0xff {
		return uint64(r)
	}
	return 0x01000000 | uint64(r)
}
