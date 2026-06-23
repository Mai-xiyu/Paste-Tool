//go:build !windows && !darwin && !linux

package platform

import "context"

type unsupportedDriver struct{}

func NewDriver() Driver {
	return unsupportedDriver{}
}

func (unsupportedDriver) Name() string { return "unsupported" }

func (unsupportedDriver) SendRune(context.Context, rune) error {
	return ErrUnsupported
}

func (unsupportedDriver) NotifyStart() error { return nil }

func (unsupportedDriver) NotifyError() error { return nil }

func (unsupportedDriver) Check(context.Context) []Check {
	return []Check{
		RuntimeCheck(),
		{Name: "input", Status: StatusUnsupported, Detail: "supported platforms are Windows, macOS, and Linux X11"},
	}
}
