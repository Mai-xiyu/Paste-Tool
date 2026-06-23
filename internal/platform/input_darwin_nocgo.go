//go:build darwin && !cgo

package platform

import "context"

type darwinNoCGODriver struct{}

func NewDriver() Driver {
	return darwinNoCGODriver{}
}

func (darwinNoCGODriver) Name() string { return "macos-coregraphics-disabled" }

func (darwinNoCGODriver) SendRune(context.Context, rune) error {
	return ErrUnsupported
}

func (darwinNoCGODriver) NotifyStart() error { return nil }

func (darwinNoCGODriver) NotifyError() error { return nil }

func (darwinNoCGODriver) Check(context.Context) []Check {
	return []Check{
		RuntimeCheck(),
		{Name: "input", Status: StatusUnsupported, Detail: "macOS input injection requires cgo"},
	}
}
