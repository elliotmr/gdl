package video

type Color struct {
	r,g,b,a uint8
}

type Palette struct {
	colors []Color
	version uint32
}

type PixelFormat struct {
	format uint32
	palette *Palette
	bitsPerPixel, bytesPerPixel uint8
	rMask, gMask, bMask, aMask uint32
	rLoss, gLoss, bLoss, aLoss uint8
	rShift, gShift, bShift, aShift uint8
	next *PixelFormat
}
