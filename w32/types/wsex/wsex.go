package wsex

type ExtendedWindowStyle uint32

const (
	DlgModalFrame    ExtendedWindowStyle = 0X00000001
	NoParentNotify   ExtendedWindowStyle = 0X00000004
	TopMost          ExtendedWindowStyle = 0X00000008
	AcceptFiles      ExtendedWindowStyle = 0X00000010
	Transparent      ExtendedWindowStyle = 0X00000020
	MDIChild         ExtendedWindowStyle = 0X00000040
	ToolWindow       ExtendedWindowStyle = 0X00000080
	WindowEdge       ExtendedWindowStyle = 0X00000100
	ClientEdge       ExtendedWindowStyle = 0X00000200
	ContextHelp      ExtendedWindowStyle = 0X00000400
	Right            ExtendedWindowStyle = 0X00001000
	Left             ExtendedWindowStyle = 0X00000000
	RTLReading       ExtendedWindowStyle = 0X00002000
	LTRReading       ExtendedWindowStyle = 0X00000000
	LeftScrollbar    ExtendedWindowStyle = 0X00004000
	RightScrollbar   ExtendedWindowStyle = 0X00000000
	ControlParent    ExtendedWindowStyle = 0X00010000
	StaticEdge       ExtendedWindowStyle = 0X00020000
	AppWindow        ExtendedWindowStyle = 0X00040000
	OverlappedWindow ExtendedWindowStyle = 0X00000100 | 0X00000200
	PaletteWindow    ExtendedWindowStyle = 0X00000100 | 0X00000080 | 0X00000008
	Layered          ExtendedWindowStyle = 0X00080000
	NoInheritLayout  ExtendedWindowStyle = 0X00100000
	LayoutRTL        ExtendedWindowStyle = 0X00400000
	NoActivate       ExtendedWindowStyle = 0X08000000
)
