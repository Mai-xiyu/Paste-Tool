package platform

import (
	"context"
	"errors"
	"time"

	"github.com/Mai-xiyu/Paste-Tool/internal/portal"
)

type waylandDriver struct{ session *portal.Session }

func (*waylandDriver) Name() string { return "linux-wayland-portal" }
func (d *waylandDriver) Prepare(ctx context.Context) error {
	if d.session != nil {
		return errors.New("a Wayland paste session is already active")
	}
	var err error
	d.session, err = portal.Open(ctx)
	return err
}
func (d *waylandDriver) Close() error {
	if d.session == nil {
		return nil
	}
	err := d.session.Close()
	d.session = nil
	return err
}
func (d *waylandDriver) SendRune(ctx context.Context, r rune) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d.session == nil {
		return errors.New("Wayland input requires an authorized RemoteDesktop session")
	}
	return d.session.SendKeysym(ctx, int32(keysymForRune(r)))
}
func (*waylandDriver) NotifyStart() error { return nil }
func (*waylandDriver) NotifyError() error { return nil }
func (*waylandDriver) Check(ctx context.Context) []Check {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	check := Check{Name: "input", Status: StatusWarning, Detail: "RemoteDesktop keyboard portal available; each paste requires desktop authorization before the start delay"}
	if err := portal.Available(ctx); err != nil {
		check.Status, check.Detail = StatusUnsupported, err.Error()
	}
	return []Check{RuntimeCheck(), check,
		{Name: "hotkey", Status: StatusWarning, Detail: "X11 global hotkeys do not work across Wayland apps; use the Paste button or a GNOME custom shortcut running paste_tool paste --source clipboard"},
	}
}
