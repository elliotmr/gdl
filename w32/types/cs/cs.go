package cs

type ClassStyle uint32

const (
	VReDraw        ClassStyle = 0x00000001
	HReDraw        ClassStyle = 0x00000002
	KeyCVTWindow   ClassStyle = 0x00000004
	DblClks        ClassStyle = 0x00000008
	OwnDC          ClassStyle = 0x00000020
	ClassDC        ClassStyle = 0x00000040
	ParentDC       ClassStyle = 0x00000080
	NoKeyCVT       ClassStyle = 0x00000100
	NoClose        ClassStyle = 0x00000200
	SaveBits       ClassStyle = 0x00000800
	ByteAlignClient ClassStyle = 0x00001000
	ByteAlignWindow ClassStyle = 0x00002000
	GlobalClass     ClassStyle = 0x00004000
	IME             ClassStyle = 0x00010000
	DropShadow      ClassStyle = 0x00020000
)