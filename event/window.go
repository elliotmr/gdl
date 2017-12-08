package event

import "encoding/binary"

// Window Events
const (
	WindowStateChange = 0x200 + iota
	SysWMEvent
)

const (
	WindowNone = iota
	WindowShown
	WindowHidden
	WindowExposed
	WindowMoved
	WindowResized
	WindowSizeChanged
	WindowMinimized
	WindowMaximized
	WindowRestored
	WindowEnter
	WindowLeave
	WindowFocusGained
	WindowFocusLost
	WindowClose
	WindowTakeFocus
	WindowHitTest
)

// Window state change event data (event.window.*)
type Window Data

func (we Window) WindowID() uint32 {
	return binary.LittleEndian.Uint32(we[8:12])
}

func (we Window) Event() uint8 {
	return we[13]
}

func (we Window) Data1() int32 {
	return int32(binary.LittleEndian.Uint32(we[16:20]))
}

func (we Window) Data2() int32 {
	return int32(binary.LittleEndian.Uint32(we[20:24]))
}

func NewWindowEvent(id uint32, windowevent uint8, data1, data2 int) Data {
	we := Data{}
	binary.LittleEndian.PutUint32(we[0:4], WindowStateChange)
	binary.LittleEndian.PutUint32(we[8:12], id)
	we[13] = windowevent
	binary.LittleEndian.PutUint32(we[16:20], uint32(data1))
	binary.LittleEndian.PutUint32(we[20:24], uint32(data2))
	return we
}