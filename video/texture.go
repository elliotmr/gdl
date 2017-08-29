package video

type Point struct {
	x, y int
}

type Rect struct {
	x, y, w, h int
}

type Texture struct {
	format     uint32
	access     int
	w, h       int
	modMode    int
	blendMode  uint32
	r, g, b, a uint8
	renderer   *Renderer

	native     *Texture
	pixels     interface{}
	pitch      int
	lockedRect Rect

	driverData interface{}

	prev *Texture
	next *Texture
}
