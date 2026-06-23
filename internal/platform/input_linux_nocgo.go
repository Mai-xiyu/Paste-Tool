//go:build linux && !cgo

package platform

import (
	"context"
	"os"
)

type linuxNoCGODriver struct{}

func NewDriver() Driver {
	return linuxNoCGODriver{}
}

func (linuxNoCGODriver) Name() string { return "linux-x11-xtest-disabled" }

func (linuxNoCGODriver) SendRune(context.Context, rune) error {
	return ErrUnsupported
}

func (linuxNoCGODriver) NotifyStart() error { return nil }

func (linuxNoCGODriver) NotifyError() error { return nil }

func (linuxNoCGODriver) Check(context.Context) []Check {
	detail := "Linux X11 input injection requires cgo with X11 and Xtst development libraries"
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") != "" {
		detail = "Wayland session detected without X11 DISPLAY; synthetic input is intentionally blocked"
	}
	return []Check{
		RuntimeCheck(),
		{Name: "input", Status: StatusUnsupported, Detail: detail},
	}
}
