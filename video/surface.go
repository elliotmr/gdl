package video

type Surface struct {
	flags uint32
	format *PixelFormat
	w, h int
	pitch int

	pixels []byte
	userdata interface{}

	locked int
	lockData interface{}

	clipRect Rect
	blitMap *BlitMap
}
