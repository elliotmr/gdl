package video

import (
	"github.com/AllenDang/w32"
	"github.com/elliotmr/gdl/scancode"
)

var windowsScanCodeTable = [...]uint{
	// 0, 1, 2, 3, 4, 5, 6,	7
	// 8, 9, A, B, C, D, E, F
	scancode.Unknown, scancode.Escape, scancode.One, scancode.Two, scancode.Three, scancode.Four, scancode.Five, scancode.Six,                                           // 0
	scancode.Seven, scancode.Eight, scancode.Nine, scancode.Zero, scancode.Minus, scancode.Equals, scancode.Backspace, scancode.Tab,                                     // 0
	scancode.Q, scancode.W, scancode.E, scancode.R, scancode.T, scancode.Y, scancode.U, scancode.I,                                                                      // 1
	scancode.O, scancode.P, scancode.LeftBracket, scancode.RightBracket, scancode.Return, scancode.LCtrl, scancode.A, scancode.S,                                        // 1
	scancode.D, scancode.F, scancode.G, scancode.H, scancode.J, scancode.K, scancode.L, scancode.Semicolon,                                                              // 2
	scancode.Apostrophe, scancode.Grave, scancode.LShift, scancode.Backslash, scancode.Z, scancode.X, scancode.C, scancode.V,                                            // 2
	scancode.B, scancode.N, scancode.M, scancode.Comma, scancode.Period, scancode.Slash, scancode.RShift, scancode.Printscreen,                                          // 3
	scancode.LAlt, scancode.Space, scancode.Capslock, scancode.F1, scancode.F2, scancode.F3, scancode.F4, scancode.F5,                                                   // 3
	scancode.F6, scancode.F7, scancode.F8, scancode.F9, scancode.F10, scancode.NumLockClear, scancode.ScrollLock, scancode.Home,                                         // 4
	scancode.Up, scancode.PageUp, scancode.KPMinus, scancode.Left, scancode.KP5, scancode.Right, scancode.KPPlus, scancode.End,                                          // 4
	scancode.Down, scancode.PageDown, scancode.Insert, scancode.Delete, scancode.Unknown, scancode.Unknown, scancode.NonUSBackslash, scancode.F11,                       // 5
	scancode.F12, scancode.Pause, scancode.Unknown, scancode.LGui, scancode.RGui, scancode.Application, scancode.Unknown, scancode.Unknown,                              // 5
	scancode.Unknown, scancode.Unknown, scancode.Unknown, scancode.Unknown, scancode.F13, scancode.F14, scancode.F15, scancode.F16,                                      // 6
	scancode.F17, scancode.F18, scancode.F19, scancode.Unknown, scancode.Unknown, scancode.Unknown, scancode.Unknown, scancode.Unknown,                                  // 6
	scancode.International2, scancode.Unknown, scancode.Unknown, scancode.International1, scancode.Unknown, scancode.Unknown, scancode.Unknown, scancode.Unknown,        // 7
	scancode.Unknown, scancode.International4, scancode.Unknown, scancode.International5, scancode.Unknown, scancode.International3, scancode.Unknown, scancode.Unknown, // 7
}

func windowsScanCodeToGDLScanCode(lParam w32.LPARAM, wParam w32.WPARAM) uint {
	nScanCode := (lParam >> 16) & 0xFF
	if nScanCode == 0 || nScanCode == 0x45 {
		switch wParam {
		case w32.VK_CLEAR:
			return scancode.Clear
		case w32.VK_MODECHANGE:
			return scancode.Mode
		case w32.VK_SELECT:
			return scancode.Select
		case w32.VK_EXECUTE:
			return scancode.Execute
		case w32.VK_HELP:
			return scancode.Help
		case w32.VK_PAUSE:
			return scancode.Pause
		case w32.VK_NUMLOCK:
			return scancode.NumLockClear

		case w32.VK_F13:
			return scancode.F13
		case w32.VK_F14:
			return scancode.F14
		case w32.VK_F15:
			return scancode.F15
		case w32.VK_F16:
			return scancode.F16
		case w32.VK_F17:
			return scancode.F17
		case w32.VK_F18:
			return scancode.F18
		case w32.VK_F19:
			return scancode.F19
		case w32.VK_F20:
			return scancode.F20
		case w32.VK_F21:
			return scancode.F21
		case w32.VK_F22:
			return scancode.F22
		case w32.VK_F23:
			return scancode.F23
		case w32.VK_F24:
			return scancode.F24

		case w32.VK_OEM_NEC_EQUAL:
			return scancode.KPEquals
		case w32.VK_BROWSER_BACK:
			return scancode.ACBack
		case w32.VK_BROWSER_FORWARD:
			return scancode.ACForward
		case w32.VK_BROWSER_REFRESH:
			return scancode.ACRefresh
		case w32.VK_BROWSER_STOP:
			return scancode.ACStop
		case w32.VK_BROWSER_SEARCH:
			return scancode.ACSearch
		case w32.VK_BROWSER_FAVORITES:
			return scancode.ACBookmarks
		case w32.VK_BROWSER_HOME:
			return scancode.ACHome
		case w32.VK_VOLUME_MUTE:
			return scancode.AudioMute
		case w32.VK_VOLUME_DOWN:
			return scancode.VolumeDown
		case w32.VK_VOLUME_UP:
			return scancode.VolumeUp

		case w32.VK_MEDIA_NEXT_TRACK:
			return scancode.AudioNext
		case w32.VK_MEDIA_PREV_TRACK:
			return scancode.AudioPrev
		case w32.VK_MEDIA_STOP:
			return scancode.AudioStop
		case w32.VK_MEDIA_PLAY_PAUSE:
			return scancode.AudioPlay
		case w32.VK_LAUNCH_MAIL:
			return scancode.Mail
		case w32.VK_LAUNCH_MEDIA_SELECT:
			return scancode.MediaSelect

		case w32.VK_OEM_102:
			return scancode.NonUSBackslash

		case w32.VK_ATTN:
			return scancode.SysReq
		case w32.VK_CRSEL:
			return scancode.CRSel
		case w32.VK_EXSEL:
			return scancode.EXSel
		case w32.VK_OEM_CLEAR:
			return scancode.Clear

		case w32.VK_LAUNCH_APP1:
			return scancode.App1
		case w32.VK_LAUNCH_APP2:
			return scancode.App2

		default:
			return scancode.Unknown

		}
	}

	if nScanCode > 127 {
		return scancode.Unknown
	}

	code := windowsScanCodeTable[nScanCode]
	bIsExtended := (lParam & (1 << 24)) != 0
	if !bIsExtended {
		switch code {
		case scancode.Home:
			return scancode.KP7
		case scancode.Up:
			return scancode.KP8
		case scancode.PageUp:
			return scancode.KP9
		case scancode.Left:
			return scancode.KP4
		case scancode.Right:
			return scancode.KP6
		case scancode.End:
			return scancode.KP1
		case scancode.Down:
			return scancode.KP2
		case scancode.PageDown:
			return scancode.KP3
		case scancode.Insert:
			return scancode.KP0
		case scancode.Delete:
			return scancode.KPPeriod
		case scancode.Printscreen:
			return scancode.KPMultiply
		}
	} else {
		switch code {
		case scancode.Return:
			return scancode.KPEnter
		case scancode.LAlt:
			return scancode.RAlt
		case scancode.LCtrl:
			return scancode.RCtrl
		case scancode.Slash:
			return scancode.KPDivide
		case scancode.Capslock:
			return scancode.KPPlus
		}
	}
	return code
}
