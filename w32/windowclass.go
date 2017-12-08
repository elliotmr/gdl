package w32

import (
	"syscall"
	"unsafe"

	"github.com/elliotmr/gdl/w32/types/cs"
	"github.com/pkg/errors"
	"github.com/elliotmr/gdl/w32/types/wm"
)

var (
	procRegisterClassEx = moduser32.NewProc("RegisterClassExW")
	procUnregisterClass = moduser32.NewProc("UnregisterClassW")
	procCreateWindowEx  = moduser32.NewProc("CreateWindowExW")
	procDefWindowProc   = moduser32.NewProc("DefWindowProcW")
	procPostQuitMessage  = moduser32.NewProc("PostQuitMessage")
)

// http://msdn.microsoft.com/en-us/library/windows/desktop/ms633577.aspx
type wndclassex struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   syscall.Handle
	icon       syscall.Handle
	cursor     syscall.Handle
	background syscall.Handle
	menuName   *uint16
	className  *uint16
	iconSm     syscall.Handle
}

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms633574(v=vs.85).aspx
// NOTE: No support for extra bytes, use window properties instead
type WindowClass struct {
	Name       string
	MenuName   string
	Style      cs.ClassStyle
	Icon       *Icon
	IconSm     *Icon
	Cursor     *Cursor
	Background *Brush

	windows []*Window
	atom    uintptr
	system  bool
}

var WindowClassButton *WindowClass = &WindowClass{Name: "Button", system: true}

func (wc *WindowClass) Register() error {
	if wc.system {
		return errors.New("cannot register a system class")
	}
	className, err := syscall.UTF16PtrFromString(wc.Name)
	if err != nil {
		return errors.Wrap(err, "invalid class name")
	}
	menuName, err := syscall.UTF16PtrFromString(wc.MenuName)
	if err != nil {
		return errors.Wrap(err, "invalid class name")
	}

	wcex := wndclassex{
		style:      uint32(wc.Style),
		instance:   inst.handle(),
		wndProc:    syscall.NewCallback(wc.process),
		icon:       wc.Icon.handle(),
		cursor:     wc.Cursor.handle(),
		background: wc.Background.handle(),
		menuName:   menuName,
		className:  className,
		iconSm:     wc.IconSm.handle(),
	}
	wcex.size = uint32(unsafe.Sizeof(wcex))
	r0, _, el := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wcex)))
	if r0 == 0 {
		if el.(syscall.Errno) != 0 {
			return el
		} else {
			return syscall.EINVAL
		}
	}
	wc.atom = r0
	return nil
}

func (wc *WindowClass) UnRegister() error {
	if wc.system {
		return errors.New("cannot unregister a system class")
	}
	r0, _, el := procUnregisterClass.Call(
		uintptr(wc.atom),
		uintptr(inst.handle()))
	if r0 == 0 {
		if el.(syscall.Errno) != 0 {
			return el
		} else {
			return syscall.EINVAL
		}
	}
	return nil
}

func (wc *WindowClass) New(handler WindowHandler, props WindowProps) (*Window, error) {
	if wc == nil {
		return nil, errors.New("window class was not registered")
	}
	name, err := syscall.UTF16PtrFromString(props.Name)
	if err != nil {
		return nil, errors.Wrap(err, "invalid window name")
	}

	w := &Window{
		class:   wc.atom,
		name:    name,
		handler: handler,
		props:   props,
	}
	wc.windows = append(wc.windows, w)
	return w, nil
}

type windowWrapper struct {
	h      syscall.Handle
	window Window
}

func (wc *WindowClass) process(hwnd syscall.Handle, uMsg uint32, wParam, lParam uintptr) uintptr {
	// TODO: change to a search method.
find_window:
	for _, w := range wc.windows {
		if hwnd == w.handle() {
			switch uMsg {
			case wm.Close:
				procDestroyWindow.Call(uintptr(hwnd))
				w.Close()
				return 0
			case wm.Destroy:
				procPostQuitMessage.Call(0)
			default:
				handled, ret := w.handler.OnMessage(uMsg, wParam, lParam)
				if handled {
					return ret
				}
				break find_window
			}
		}
	}
	ret, _, _ := procDefWindowProc.Call(
		uintptr(hwnd),
		uintptr(uMsg),
		wParam,
		lParam,
	)
	return ret
}
