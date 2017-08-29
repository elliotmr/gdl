package video

import "github.com/pkg/errors"

// videoDisplay defines the the structure that corresponds to a physical monitor attached to the system.
type videoDisplay struct  {
	name string
	maxDisplayModes int
	numDisplayModes int
	modes []DisplayMode
	desktopMode DisplayMode
	currentMode DisplayMode

	fullscreenWindow *Window
	device           videoDevice

	driverdata interface{}
}

func getDisplayBounds(displayIndex int) (Rect, error) {
	if displayIndex >= len(this.data().displays) {
		return Rect{}, errors.New("invalid display index")
	}

	display := this.data().displays[displayIndex]
	rect, err := this.getDisplayBounds(display)
	if err != nil {
		return rect, errors.Wrap(err, "driver get display bounds failed")
	}

	if displayIndex == 0 {
		rect.x = 0
		rect.y = 0
	} else {
		r, err := getDisplayBounds(displayIndex-1)
		if err != nil {
			return rect, err  // don't wrap recursive call
		}
		rect.x += r.w
	}
	rect.w = display.currentMode.w
	rect.h = display.currentMode.h
	return rect, nil
}