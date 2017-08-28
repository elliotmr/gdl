package event

import "encoding/binary"

// Keyboard Events
const (
	KeyDown = 0x300 + iota
	KeyUp
	TextEditing
	TextInput
	KeyMapChanged
)

// Keyboard button event structure (event.key.*)
type KeyboardEvent Data

func (ke KeyboardEvent) WindowID() uint32 {
	return binary.LittleEndian.Uint32(ke[8:12])
}

func (ke KeyboardEvent) State() uint8 {
	return ke[12]
}
func (ke KeyboardEvent) Repeat() uint8 {
	return ke[13]
}

func (ke KeyboardEvent) ScanCode() uint32 {
	return binary.LittleEndian.Uint32(ke[16:20])
}

func (ke KeyboardEvent) KeyCode() int32 {
	return int32(binary.LittleEndian.Uint32(ke[20:24]))
}

func (ke KeyboardEvent) Mod() uint16 {
	return binary.LittleEndian.Uint16(ke[24:26])
}
