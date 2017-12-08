package event

import "encoding/binary"

// Mouse Events
const (
	MouseMotion = 0x400 + iota
	MouseButtonDown
	MouseButtonUp
	MouseWheel
)

var M *Mouse

func init() {
	M = &Mouse{}
}

// Keyboard button event structure (event.key.*)
type MouseButton Data

func (mbe MouseButton) WindowID() uint32 {
	return binary.LittleEndian.Uint32(mbe[8:12])
}

func (mbe MouseButton) Which() uint32 {
	return binary.LittleEndian.Uint32(mbe[12:16])
}

func (mbe MouseButton) Button() uint8 {
	return mbe[16]
}

func (mbe MouseButton) State() uint8 {
	return mbe[17]
}

func (mbe MouseButton) Clicks() uint8 {
	return mbe[18]
}

func (mbe MouseButton) X() int32 {
	return int32(binary.LittleEndian.Uint32(mbe[20:24]))
}

func (mbe MouseButton) Y() int32 {
	return int32(binary.LittleEndian.Uint16(mbe[24:26]))
}

type Mouse struct {

}

func (m *Mouse) FreeCursor() {

}