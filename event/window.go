package event

import "encoding/binary"

// Window Events
const (
	WindowStateChange = 0x200 + iota
	SysWMEvent
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