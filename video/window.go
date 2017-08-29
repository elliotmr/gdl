package video

import (
	"github.com/pkg/errors"
	"sync/atomic"
)

const (
	WindowFullscreen        = 1 << iota // fullscreen window
	WindowOpenGL                        // window usable with OpenGL context
	WindowShown                         // window is visible
	WindowHidden                        // window is not visible
	WindowBorderless                    // no window decoration
	WindowResizable                     // window can be resized
	WindowMinimized                     // window is minimized
	WindowMaximized                     // window is maximized
	WindowInputGrabbed                  // window has grabbed input focus
	WindowInputFocus                    // window has input focus
	WindowMouseFocus                    // window has mouse focus
	WindowForeign                       // window not created by SDL
	WindowFullscreenDesktop = 1<<iota | WindowFullscreen
	WindowAllowHighDPI      // window should be created in high-DPI mode if supported
	WindowMouseCaputure     // window has mouse captured (unrelated to INPUT_GRABBED)
	WindowAlwaysOnTop       // window should always be above others
	WindowSkipTaskbar       // window should not be added to the taskbar
	WindowUtility           // window should be treated as a utility window
	WindowTooltip           // window should be treated as a tooltip
	WindowPopupMenu         // window should be treated as a popup menu
)

const WindowPosUndefined = 0x1FFF0000
const WindowPosCentered = 0x2FFF0000

func WindowPosIsUndefined(x int) bool {
	return x | WindowPosUndefined == WindowPosUndefined
}

func WindowPosIsCentered(x int) bool {
	return x | WindowPosCentered == WindowPosCentered
}

type DisplayMode struct {
	format      uint32
	w, h        int
	refreshRate int
	driverData  interface{}
}

type windowShapeMode struct {
}

type windowShaper struct {
	window       *Window
	userX, userY uint32
	mode         windowShapeMode
	hasShape     bool
	driverdata   interface{}
}

type Window struct {
	magic               uint8
	id                  uint32
	title               string
	icon                *Surface
	x, y                int
	w, h                int
	minW, minH          int
	maxW, maxH          int
	flags               uint32
	lastFullscreenFlags uint32

	windowed       Rect // stored position and size for windowed mode
	fullscreenMode DisplayMode

	opacity    float32
	brightness float32

	gamma      []uint16
	savedGamma []uint16

	surface      *Surface
	surfaceValid bool

	isHiding     bool
	isDestroying bool
	isDropping   bool

	shaper windowShaper

	// TODO(mde): add window hit test

	data map[string]interface{}

	prev *Window
	next *Window
}

func CreateWindow(title string, x, y, w, h int, flags uint32) (*Window, error) {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}

	if (w > 16384) || (h > 16384) {
		return nil, errors.New("window too large")
	}

	// TODO(mde): check for OpengGL support with flag WindowOpenGL
	// TODO(mde): disable high DPI if hint high spi disabled is set
	window := &Window{
		magic: this.data().windowMagic,
		x: x,
		y: y,
		w: w,
		h: h,
	}

	window.id = atomic.AddUint32(&(this.data().nextObjectID), 1)

	window.windowed.x = window.x
	window.windowed.y = window.y
	window.windowed.w = window.w
	window.windowed.h = window.h

	// TODO(mde): Lots more

	return window, nil

}


