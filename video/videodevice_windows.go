// +build windows

package video

import (
	"github.com/elliotmr/gdl/w32"
	"github.com/elliotmr/gdl/w32/types/ws"
	"github.com/elliotmr/gdl/event"
	"github.com/pkg/errors"
	"fmt"
)

func init() {
	this = &winVideoDevice{
		deviceData: &videoDeviceData{

		},
	}
	registerApp("", 0)
}

type winVideoDevice struct {
	deviceData *videoDeviceData
}

func (wvd *winVideoDevice) Pump(q *event.Queue) {
	panic("implement me")
}

func (wvd *winVideoDevice) getDisplayModes(display *videoDisplay) {
	panic("implement me")
}

func (wvd *winVideoDevice) setDisplayMode(display *videoDisplay, mode *DisplayMode) {
	panic("implement me")
}

const styleBasic = ws.ClipChildren | ws.ClipSiblings
const styleFullscreen = ws.Popup
const styleBorderless = ws.Popup
const styleNormal = ws.Overlapped | ws.Caption | ws.SysMenu | ws.MinimizeBox
const styleResizable = ws.ThickFrame | ws.MaximizeBox
const styleMask = styleFullscreen | styleBorderless | styleNormal | styleResizable

func getWindowStyle(window *Window) ws.WindowStyle {
	var style ws.WindowStyle
	if window.flags & WindowFullscreen > 0 {
		style |= styleFullscreen
	} else {
		if window.flags & WindowBorderless > 0 {
			style |= styleBorderless
		} else {
			style |= styleNormal
		}
		if window.flags & WindowResizable > 0 {
			style |= styleResizable
		}
	}
	return style
}

func (wvd *winVideoDevice) createWindow(window *Window) error {
	var style ws.WindowStyle = styleBasic | getWindowStyle(window)

	props := w32.WindowProps{
		Name: window.title,
		X: int64(window.x),
		Y: int64(window.y),
		Width: int64(window.w),
		Height: int64(window.h),
		Style: style | ws.Visible,
	}
	handler := &eventHandler{window: window}
	fmt.Println("creating window")
	w, err := GDLAppClass.New(handler, props)
	go func() {
		err = w.Run()
		if err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return errors.Wrap(err, "unable to create window")
	}
	window.data["native"] = w // store so it isn't garbage collected.
	return nil
}

func (wvd *winVideoDevice) createWindowFrom(window *Window, data interface{}) error {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowTitle(windows *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowIcon(window *Window, icon *Surface) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowPosition(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowMinimumSize(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowMaximumSize(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) getWindowBordersSize(window *Window) (int, int, int, int, error) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowOpacity(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowModalFor(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowInputFocus(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) showWindow(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) hideWindow(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) raiseWindow(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) maximizeWindow(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) minimizeWindow(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) restoreWindow(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowBordered(window *Window, bordered bool) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowResizable(window *Window, resizeable bool) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowFullscreen(window *Window, display *videoDisplay, fullscreen bool) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowGammaRamp(window *Window, ramp []uint16) {
	panic("implement me")
}

func (wvd *winVideoDevice) getWindowGammaRamp(window *Window, ramp []uint16) {
	panic("implement me")
}

func (wvd *winVideoDevice) setWindowGrab(window *Window, grabbed bool) {
	panic("implement me")
}

func (wvd *winVideoDevice) destroyWindow(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) createWindowFramebuffer(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) updateWindowFramebuffer(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) destroyWindowFramebuffer(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) onWindowEnter(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) glLoadLibrary(path string) {
	panic("implement me")
}

func (wvd *winVideoDevice) glGetProcAddress(proc string) {
	panic("implement me")
}

func (wvd *winVideoDevice) glUnloadLibrary() {
	panic("implement me")
}

func (wvd *winVideoDevice) glCreateContext() {
	panic("implement me")
}

func (wvd *winVideoDevice) glMakeCurrent() {
	panic("implement me")
}

func (wvd *winVideoDevice) glSetSwapInterval(interval int) {
	panic("implement me")
}

func (wvd *winVideoDevice) glGetSwapInterval() {
	panic("implement me")
}

func (wvd *winVideoDevice) glSwapWindow(window *Window) {
	panic("implement me")
}

func (wvd *winVideoDevice) glDeleteContext() {
	panic("implement me")
}

func (wvd *winVideoDevice) data() *videoDeviceData {
	return wvd.deviceData
}

func (wvd *winVideoDevice) init() {
	panic("implement me")
}

func (wvd *winVideoDevice) quit() {
	panic("implement me")
}

func (wvd *winVideoDevice) getDisplayBounds(display *videoDisplay) (Rect, error) {
	panic("implement me")
}

func (wvd *winVideoDevice) getDisplayDPI(display *videoDisplay) (float32, float32, float32, error) {
	panic("implement me")
}

func (wvd *winVideoDevice) getDisplayUsableBounds(display *videoDisplay) (Rect, error) {
	panic("implement me")
}

