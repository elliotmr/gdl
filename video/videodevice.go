package video

import (
	"github.com/elliotmr/gdl/event"
)

// I don't really like this, but it will make the porting much easier.
var this videoDevice

type videoDevice interface {
	event.Pumper

	data() *videoDeviceData
	init()
	quit()

	// Display Functions
	getDisplayBounds(display *videoDisplay) (Rect, error)
	getDisplayDPI(display *videoDisplay) (float32, float32, float32, error)
	getDisplayUsableBounds(display *videoDisplay) (Rect, error)
	getDisplayModes(display *videoDisplay)
	setDisplayMode(display *videoDisplay, mode *DisplayMode)

	// Window Functions
	createWindow(window *Window) error
	createWindowFrom(window *Window, data interface{}) error
	setWindowTitle(windows *Window)
	setWindowIcon(window *Window, icon *Surface)
	setWindowPosition(window *Window)
	setWindowMinimumSize(window *Window)
	setWindowMaximumSize(window *Window)
	getWindowBordersSize(window *Window) (int, int, int, int, error)
	setWindowOpacity(window *Window)
	setWindowModalFor(window *Window)
	setWindowInputFocus(window *Window)
	showWindow(window *Window)
	hideWindow(window *Window)
	raiseWindow(window *Window)
	maximizeWindow(window *Window)
	minimizeWindow(window *Window)
	restoreWindow(window *Window)
	setWindowBordered(window *Window, bordered bool)
	setWindowResizable(window *Window, resizeable bool)
	setWindowFullscreen(window *Window, display *videoDisplay, fullscreen bool)
	setWindowGammaRamp(window *Window, ramp []uint16)
	getWindowGammaRamp(window *Window, ramp []uint16)
	setWindowGrab(window *Window, grabbed bool)
	destroyWindow(window *Window)
	createWindowFramebuffer(window *Window)
	updateWindowFramebuffer(window *Window)
	destroyWindowFramebuffer(window *Window)
	onWindowEnter(window *Window)

	// OpenGL support
	glLoadLibrary(path string)
	glGetProcAddress(proc string)
	glUnloadLibrary()
	glCreateContext() // not required
	glMakeCurrent() // combines creation and make current
	glSetSwapInterval(interval int)
	glGetSwapInterval()
	glSwapWindow(window *Window)
	glDeleteContext() // not required
}

type videoDeviceData struct {
	name               string
	suspendScreenSaver bool
	displays []*videoDisplay
	windows []*Window
	grabbedWindow *Window
	windowMagic uint8
	nextObjectID uint32
	clipboardText string
}