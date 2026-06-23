//go:build windows

package platform

import (
	"context"
	"fmt"
	"syscall"
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
	user32          = syscall.NewLazyDLL("user32.dll")
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procSendInput   = user32.NewProc("SendInput")
	procMessageBeep = user32.NewProc("MessageBeep")
	procBeep        = kernel32.NewProc("Beep")
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
	if r == '\n' {
		return sendInputs([]input{
			{Type: inputKeyboard, Ki: keyboardInput{WVk: vkReturn}},
			{Type: inputKeyboard, Ki: keyboardInput{WVk: vkReturn, DwFlags: keyEventKeyUp}},
		})
	}
	units := utf16.Encode([]rune{r})
	for _, unit := range units {
		if err := sendInputs([]input{
			{Type: inputKeyboard, Ki: keyboardInput{WScan: unit, DwFlags: keyEventUnicode}},
			{Type: inputKeyboard, Ki: keyboardInput{WScan: unit, DwFlags: keyEventUnicode | keyEventKeyUp}},
		}); err != nil {
			return err
		}
	}
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
