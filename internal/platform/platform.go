package platform

import (
	"context"
	"errors"
	"runtime"
)

var ErrUnsupported = errors.New("platform unsupported")

type Status string

const (
	StatusOK          Status = "ok"
	StatusWarning     Status = "warning"
	StatusUnsupported Status = "unsupported"
	StatusError       Status = "error"
)

type Check struct {
	Name   string
	Status Status
	Detail string
}

type Driver interface {
	Name() string
	SendRune(context.Context, rune) error
	NotifyStart() error
	NotifyError() error
	Check(context.Context) []Check
}

func RuntimeCheck() Check {
	return Check{
		Name:   "runtime",
		Status: StatusOK,
		Detail: runtime.GOOS + "/" + runtime.GOARCH,
	}
}
