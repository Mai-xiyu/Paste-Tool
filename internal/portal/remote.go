// Package portal implements the keyboard-only XDG RemoteDesktop session.
package portal

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	busName          = "org.freedesktop.portal.Desktop"
	desktopPath      = dbus.ObjectPath("/org/freedesktop/portal/desktop")
	remoteInterface  = "org.freedesktop.portal.RemoteDesktop"
	requestInterface = "org.freedesktop.portal.Request"
)

type Session struct {
	conn *dbus.Conn
	path dbus.ObjectPath
}

func Available(ctx context.Context) error {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return fmt.Errorf("connect session bus: %w", err)
	}
	defer conn.Close()
	var value dbus.Variant
	err = conn.Object(busName, desktopPath).CallWithContext(ctx,
		"org.freedesktop.DBus.Properties.Get", 0, remoteInterface, "AvailableDeviceTypes").Store(&value)
	if err != nil {
		return fmt.Errorf("RemoteDesktop portal unavailable (install xdg-desktop-portal and the GNOME backend): %w", err)
	}
	devices, ok := value.Value().(uint32)
	if !ok || devices&1 == 0 {
		return errors.New("RemoteDesktop portal does not offer keyboard input")
	}
	return nil
}

func Open(ctx context.Context) (*Session, error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect session bus: %w", err)
	}
	s := &Session{conn: conn}
	// Predict the session path so cancellation can also clean up CreateSession.
	token := "paste_tool_" + rand.Text()
	s.path = dbus.ObjectPath(string(desktopPath) + "/session/" + sender(conn) + "/" + token)
	results, err := request(ctx, conn, "CreateSession", map[string]dbus.Variant{
		"session_handle_token": dbus.MakeVariant(token),
	})
	if err == nil {
		var path string
		path, err = sessionHandle(results)
		if err == nil {
			s.path = dbus.ObjectPath(path)
		}
	}
	if err == nil {
		_, err = request(ctx, conn, "SelectDevices", map[string]dbus.Variant{
			"types": dbus.MakeVariant(uint32(1)),
		}, s.path)
	}
	if err == nil {
		results, err = request(ctx, conn, "Start", nil, s.path, "")
		if err == nil {
			err = keyboardGranted(results)
		}
	}
	if err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

func (s *Session) SendKeysym(ctx context.Context, symbol int32) error {
	if s.conn == nil {
		return errors.New("remote desktop session is closed")
	}
	obj := s.conn.Object(busName, desktopPath)
	args := []interface{}{s.path, map[string]dbus.Variant{}, symbol, uint32(1)}
	err := obj.CallWithContext(ctx, remoteInterface+".NotifyKeyboardKeysym", 0, args...).Err
	// A canceled call can still have reached the compositor. Always release it.
	releaseCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	args[3] = uint32(0)
	releaseErr := obj.CallWithContext(releaseCtx, remoteInterface+".NotifyKeyboardKeysym", 0, args...).Err
	return errors.Join(err, releaseErr)
}

func (s *Session) Close() error {
	if s.conn == nil {
		return nil
	}
	closeObject(s.conn, s.path, "org.freedesktop.portal.Session")
	err := s.conn.Close()
	s.conn = nil
	return err
}

func closeObject(conn *dbus.Conn, path dbus.ObjectPath, iface string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = conn.Object(busName, path).CallWithContext(ctx, iface+".Close", 0).Err
}

func sender(conn *dbus.Conn) string {
	return strings.ReplaceAll(strings.TrimPrefix(conn.Names()[0], ":"), ".", "_")
}

func request(ctx context.Context, conn *dbus.Conn, method string, options map[string]dbus.Variant, args ...interface{}) (map[string]dbus.Variant, error) {
	if options == nil {
		options = make(map[string]dbus.Variant)
	}
	token := "paste_tool_" + rand.Text()
	options["handle_token"] = dbus.MakeVariant(token)
	path := dbus.ObjectPath(string(desktopPath) + "/request/" + sender(conn) + "/" + token)
	signals := make(chan *dbus.Signal, 16)
	conn.Signal(signals)
	defer conn.RemoveSignal(signals)
	match := []dbus.MatchOption{
		dbus.WithMatchSender(busName),
		dbus.WithMatchInterface(requestInterface),
		dbus.WithMatchMember("Response"),
		dbus.WithMatchPathNamespace(dbus.ObjectPath(string(desktopPath) + "/request")),
	}
	if err := conn.AddMatchSignalContext(ctx, match...); err != nil {
		return nil, err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = conn.RemoveMatchSignalContext(cleanup, match...)
	}()
	var returned dbus.ObjectPath
	if err := conn.Object(busName, desktopPath).CallWithContext(ctx, remoteInterface+"."+method, 0, append(args, options)...).Store(&returned); err != nil {
		closeObject(conn, path, requestInterface)
		return nil, fmt.Errorf("portal %s: %w", method, err)
	}
	path = returned
	for {
		select {
		case <-ctx.Done():
			closeObject(conn, path, requestInterface)
			return nil, ctx.Err()
		case signal, ok := <-signals:
			if !ok {
				return nil, errors.New("portal disconnected")
			}
			if signal.Path != path || signal.Name != requestInterface+".Response" {
				continue
			}
			results, err := response(signal.Body)
			if err != nil {
				return nil, fmt.Errorf("portal %s: %w", method, err)
			}
			return results, nil
		}
	}
}

func response(body []interface{}) (map[string]dbus.Variant, error) {
	var code uint32
	var results map[string]dbus.Variant
	if err := dbus.Store(body, &code, &results); err != nil {
		return nil, fmt.Errorf("invalid portal response: %w", err)
	}
	switch code {
	case 0:
		return results, nil
	case 1:
		return nil, errors.New("keyboard permission was canceled or denied")
	default:
		return nil, fmt.Errorf("desktop portal rejected the request (response %d)", code)
	}
}

func sessionHandle(results map[string]dbus.Variant) (string, error) {
	path, ok := results["session_handle"].Value().(string)
	if !ok || !dbus.ObjectPath(path).IsValid() || !strings.HasPrefix(path, string(desktopPath)+"/session/") {
		return "", errors.New("portal returned an invalid session_handle")
	}
	return path, nil
}

func keyboardGranted(results map[string]dbus.Variant) error {
	devices, ok := results["devices"].Value().(uint32)
	if !ok || devices&1 == 0 {
		return errors.New("desktop portal did not grant keyboard permission")
	}
	return nil
}
