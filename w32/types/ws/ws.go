package ws

type WindowStyle uint32

// Window style constants
const (
	Overlapped       WindowStyle = 0X00000000
	Popup            WindowStyle = 0X80000000
	Child            WindowStyle = 0X40000000
	Minimize         WindowStyle = 0X20000000
	Visible          WindowStyle = 0X10000000
	Disabled         WindowStyle = 0X08000000
	ClipSiblings     WindowStyle = 0X04000000
	ClipChildren     WindowStyle = 0X02000000
	Maximize         WindowStyle = 0X01000000
	Caption          WindowStyle = 0X00C00000
	Border           WindowStyle = 0X00800000
	DlgFrame         WindowStyle = 0X00400000
	VScroll          WindowStyle = 0X00200000
	HScroll          WindowStyle = 0X00100000
	SysMenu          WindowStyle = 0X00080000
	ThickFrame       WindowStyle = 0X00040000
	Group            WindowStyle = 0X00020000
	TabStop          WindowStyle = 0X00010000
	MinimizeBox      WindowStyle = 0X00020000
	MaximizeBox      WindowStyle = 0X00010000
	Tiled            WindowStyle = 0X00000000
	Iconic           WindowStyle = 0X20000000
	SizeBox          WindowStyle = 0X00040000
	OverlappedWindow WindowStyle = 0X00000000 | 0X00C00000 | 0X00080000 | 0X00040000 | 0X00020000 | 0X00010000
	PopupWindow      WindowStyle = 0X80000000 | 0X00800000 | 0X00080000
	ChildWindow      WindowStyle = 0X40000000
)