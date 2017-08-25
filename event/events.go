package event

import "encoding/binary"

const KeyReleased = 0
const KeyPressed = 1

// Application Events
const (
	Quit = 0x100 + iota
	AppTerminating
	AppLowMemory
	AppWillEnterBackground
	AppDidEnterBackground
	AppWillEnterForeground
	AppDidEnterForeground
)

// Data is the raw event data, 
type Data [56]byte

func (ed Data) Type() uint32 {
	return binary.LittleEndian.Uint32(ed[0:4])
}

func (ed Data) Timestamp() uint32 {
	return binary.LittleEndian.Uint32(ed[4:8])
}

func (ed Data) Raw() Data {
	return ed
}

type Event interface {
	Type() uint32
	Timestamp() uint32
	Raw() Data
}









// Joystick Events
const (
	JoyAxisMotion = 0x600 + iota
	JoyBallMotion
	JoyHatMotion
	JoyButtonDown
	JoyButtonUp
	JoyDeviceAdded
	JoyDeviceRemoved
)

// Game Controller Events
const (
	ControllerAxisMotion = 0x650 + iota
	ControllerButtonDown
	ControllerButtonUp
	ControllerDeviceAdded
	ControllerDeviceRemoved
	ControllerDeviceRemapped
)

// Touch Events
const (
	FingerDown = 0x700 + iota
	FingerUp
	FingerMotion
)

// Gesture Events
const (
	DollarGesture = 0x800 + iota
	DollarRecord
	MultiGesture
)

// Clipboard Events
const (
	ClipboardUpdate = 0x900 + iota
)

// Drag and Drop Events
const (
	DropFile = 0x1000 + iota
	DropText
	DropBegin
	DropComplete
)

// Audio Hotplug Events
const (
	AudioDeviceAdded = 0x1100 + iota
	AudioDeviceRemoved
)

// Render Events
const (
	RenderTargetsReset = 0x2000 + iota
	RenderDeviceReset
)

const (
	UserEvent = 0x8000
	LastEvent = 0xFFFF
)


