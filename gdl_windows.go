// +build windows

package gdl

import (
	"github.com/AllenDang/w32"
	"github.com/pkg/errors"
	"golang.org/x/sys/windows"
	"syscall"
)

const (
	NULL                       = 0
	ERROR_CLASS_ALREADY_EXISTS = 1410
)

// static globals
var helperWindow w32.HWND = NULL
var helperWindowClassName string = "HelperWindowInputCatcher"
var helperWindowName string = "HelperWindowInputMsgWindow"
var helperWindowClass w32.ATOM = 0

func helperWindowCreate() error {
	hInstance := w32.GetModuleHandle("")

	wce := w32.WNDCLASSEX{
		WndProc:   windows.NewCallback(w32.DefWindowProc),
		ClassName: windows.StringToUTF16Ptr(helperWindowClassName),
		Instance:  hInstance,
	}
	var err error
	helperWindowClass, err = w32.RegisterClassEx(&wce)
	if err != nil && err.(syscall.Errno) != ERROR_CLASS_ALREADY_EXISTS {
		return errors.Wrap(err, "unable to create helper window class")
	}

	helperWindow = w32.CreateWindowEx(
		0,
		windows.StringToUTF16Ptr(helperWindowClassName),
		windows.StringToUTF16Ptr(helperWindowName),
		w32.WS_OVERLAPPED,
		w32.CW_USEDEFAULT,
		w32.CW_USEDEFAULT,
		w32.CW_USEDEFAULT,
		w32.CW_USEDEFAULT,
		w32.HWND_MESSAGE,
		NULL,
		hInstance,
		nil,
	)

	if helperWindow == NULL {
		w32.UnregisterClass(
			windows.StringToUTF16Ptr(helperWindowClassName),
			hInstance,
		)
		return errors.New("unable to create helper window")
	}
	return nil
}
