package w32

import (
	"syscall"

	"github.com/elliotmr/gdl/w32/types/ws"
	"github.com/elliotmr/gdl/w32/types/wsex"
	"runtime"
	"unsafe"
	"context"
	"time"
)

var (
	procGetTickCount       = modkernel32.NewProc("GetTickCount")
	procDestroyWindow      = moduser32.NewProc("DestroyWindow")
	procDispatchMessage    = moduser32.NewProc("DispatchMessageW")
	procGetQueueStatus     = moduser32.NewProc("GetQueueStatus")
	procGetMessage         = moduser32.NewProc("GetMessageW")
	procTranslateMessage   = moduser32.NewProc("TranslateMessage")
	procAdjustWindowRectEx = moduser32.NewProc("AdjustWindowRectEx")
)

// https://msdn.microsoft.com/en-us/library/windows/desktop/ms644958(v=vs.85).aspx
type msg struct {
	hwnd    syscall.Handle
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      point
}

// https://msdn.microsoft.com/en-us/library/windows/desktop/dd162805(v=vs.85).aspx
type point struct {
	x int32
	y int32
}

type WindowHandler interface {
	OnMessage(uint32, uintptr, uintptr) (bool, uintptr)
}

const WPUseDefault = 0x80000000

type WindowProps struct {
	Name          string
	Style         ws.WindowStyle
	ExtendedStyle wsex.ExtendedWindowStyle
	X             int64
	Y             int64
	Width         int64
	Height        int64
	Parent        *Window
	Menu          *Menu
	lpParam       uintptr
}

type Window struct {
	ctx     context.Context
	cancel  context.CancelFunc
	ticker  *time.Ticker
	h       syscall.Handle
	class   uintptr
	name    *uint16
	handler WindowHandler
	props   WindowProps
}

func (w *Window) handle() syscall.Handle {
	if w == nil {
		return 0
	}
	return w.h
}

func (w *Window) Close() {
	w.cancel()
}

func (w *Window) Run() error {
	runtime.LockOSThread()
	if w.props.Style != ws.Overlapped && w.props.X != WPUseDefault && w.props.Y != WPUseDefault && w.props.Width != WPUseDefault && w.props.Height != WPUseDefault {
		rect := struct {
			left   int32
			top    int32
			right  int32
			bottom int32
		}{
			int32(w.props.X),
			int32(w.props.Y),
			int32(w.props.X + w.props.Width),
			int32(w.props.Y + w.props.Height),
		}

		ret, _, err := procAdjustWindowRectEx.Call(
			uintptr(unsafe.Pointer(&rect)),
			uintptr(w.props.Style),
			uintptr(BoolToBOOL(w.props.Menu != nil)),
			uintptr(w.props.ExtendedStyle),
		)
		if ret == 0 {
			if err.(syscall.Errno) != 0 {
				return err
			} else {
				return syscall.EINVAL
			}
		}
		w.props.X = int64(rect.left)
		w.props.Y = int64(rect.top)
		w.props.Width = int64(rect.right - rect.left)
		w.props.Height = int64(rect.bottom - rect.top)
	}

	r0, _, el := procCreateWindowEx.Call(
		uintptr(w.props.ExtendedStyle),
		w.class,
		uintptr(unsafe.Pointer(w.name)),
		uintptr(w.props.Style),
		uintptr(w.props.X),
		uintptr(w.props.Y),
		uintptr(w.props.Width),
		uintptr(w.props.Height),
		uintptr(w.props.Parent.handle()),
		uintptr(w.props.Menu.handle()),
		uintptr(inst.handle()),
		w.props.lpParam)
	if r0 == 0 {
		if el.(syscall.Errno) != 0 {
			return el
		} else {
			return syscall.EINVAL
		}
	}
	m := msg{}
	w.h = syscall.Handle(r0)
	w.ctx, w.cancel = context.WithCancel(context.Background())
	w.ticker = time.NewTicker(15 * time.Millisecond)

	// window_loop:
	for {
		select {
		case <-w.ticker.C:
			r, _, _ := procGetTickCount.Call()
			start := uint32(r)
			process_queue_loop:
			for {
				ret, _, _ := procGetQueueStatus.Call(uintptr(0xFF))
				if ret == 0 {
					break process_queue_loop // queue empty
				}
				ret, _, err := procGetMessage.Call(
					uintptr(unsafe.Pointer(&m)),
					uintptr(w.handle()),
					0, 0,
				)
				switch int32(ret) {
				case -1:
					return err
				case 0:
					return nil
				default:
					procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
					procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
				}
				if m.time - start > 0 { // break out once we have process messages at beginning of tick
					break process_queue_loop
				}
			}
			// TODO: Shift fix from SDL_windowsevents.c lines 1026-1036

		case <-w.ctx.Done():
			return nil
		}
	}
	return nil
}
