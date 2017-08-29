package video

// I don't really like this, but it will make the porting much easier.
var this videoDevice

type videoDevice interface {
	data() *videoDeviceData
	init()
	quit()

	getDisplayBounds(display *videoDisplay) (Rect, error)
	getDisplayDPI(display *videoDisplay) (float32, float32, float32, error)
	getDisplayUsableBounds(display *videoDisplay) (Rect, error)

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