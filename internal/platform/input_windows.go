//go:build windows

package platform

import (
	"context"
	"fmt"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	inputKeyboard   = 1
	keyEventKeyUp   = 0x0002
	keyEventUnicode = 0x0004
	vkReturn        = 0x0D
	messageBeepOK   = 0
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procSendInput        = user32.NewProc("SendInput")
	procMessageBeep      = user32.NewProc("MessageBeep")
	procBeep             = kernel32.NewProc("Beep")
	procGetAsyncKeyState = user32.NewProc("GetAsyncKeyState")
)

type keyboardInput struct {
	WVk         uint16
	WScan       uint16
	DwFlags     uint32
	Time        uint32
	DwExtraInfo uintptr
}

type input struct {
	Type uint32
	Ki   keyboardInput
	// INPUT contains a union sized for MOUSEINPUT, not just KEYBDINPUT.
	_ [8]byte
}

type windowsDriver struct{}

func NewDriver() Driver {
	return windowsDriver{}
}

func (windowsDriver) Name() string {
	return "windows-sendinput"
}

func (windowsDriver) SendRune(ctx context.Context, r rune) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return sendInputs(inputsForRune(r))
}

func inputsForRune(r rune) []input {
	if r == '\n' || r == '\t' {
		key := uint16(vkReturn)
		if r == '\t' {
			key = 0x09
		}
		return []input{
			input{Type: inputKeyboard, Ki: keyboardInput{WVk: key}},
			input{Type: inputKeyboard, Ki: keyboardInput{WVk: key, DwFlags: keyEventKeyUp}},
		}
	}
	units := utf16.Encode([]rune{r})
	events := make([]input, 0, 2*len(units))
	for _, unit := range units {
		events = append(events,
			input{Type: inputKeyboard, Ki: keyboardInput{WScan: unit, DwFlags: keyEventUnicode}},
			input{Type: inputKeyboard, Ki: keyboardInput{WScan: unit, DwFlags: keyEventUnicode | keyEventKeyUp}},
		)
	}
	return events
}

func (windowsDriver) Prepare(ctx context.Context) error {
	// Wait for the invoking shortcut to be released; never release the user's keys.
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		held := false
		for _, key := range []uintptr{0x10, 0x11, 0x12, 0x5b, 0x5c} {
			state, _, _ := procGetAsyncKeyState.Call(key)
			held = held || state&0x8000 != 0
		}
		if !held {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("release Ctrl, Alt, Shift and Windows keys before pasting")
		case <-tick.C:
		}
	}
}

func (windowsDriver) Close() error {
	return nil
}

func (windowsDriver) NotifyStart() error {
	_, _, _ = procMessageBeep.Call(uintptr(messageBeepOK))
	return nil
}

func (windowsDriver) NotifyError() error {
	_, _, _ = procBeep.Call(200, 200)
	return nil
}

func (windowsDriver) Check(context.Context) []Check {
	return []Check{
		RuntimeCheck(),
		{Name: "input", Status: StatusOK, Detail: "SendInput is available; elevated target windows may reject lower-integrity input through UIPI"},
	}
}

func sendInputs(inputs []input) error {
	if len(inputs) == 0 {
		return nil
	}
	sent, _, err := procSendInput.Call(
		uintptr(len(inputs)),
		uintptr(unsafe.Pointer(&inputs[0])),
		unsafe.Sizeof(input{}),
	)
	if sent != uintptr(len(inputs)) {
		if err != syscall.Errno(0) {
			return fmt.Errorf("SendInput sent %d/%d events: %w", sent, len(inputs), err)
		}
		return fmt.Errorf("SendInput sent %d/%d events; possible UIPI integrity-level block", sent, len(inputs))
	}
	return nil
}
