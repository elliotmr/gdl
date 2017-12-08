package video

import (
	"sync"
	"github.com/elliotmr/gdl/event"
	"github.com/elliotmr/gdl/w32"
	"github.com/elliotmr/gdl/w32/types/cs"
	"github.com/elliotmr/gdl/w32/types/wm"
	"fmt"
)

var GDLAppClass *w32.WindowClass

type eventHandler struct {
	window *Window
}

func (eh *eventHandler) OnMessage(uMsg uint32, wParam, lParam uintptr) (bool, uintptr) {
	if event.Q.Enabled(event.SysWMEvent) {
		// TODO: Deal with it.
	}
	switch uMsg {
	case wm.ShowWindow:
		if wParam > 0 {
			eh.window.SendEvent(event.WindowShown, 0, 0)
		} else {
			eh.window.SendEvent(event.WindowHidden, 0, 0)
		}
	case wm.Activate:
	case wm.MouseMove:

		fallthrough
	case wm.LButtonUp, wm.LButtonDown, wm.LButtonDblClk, wm.RButtonUp, wm.RButtonDown, wm.RButtonDblClk,
		 wm.MButtonUp, wm.MButtonDown, wm.MButtonDblClk, wm.XButtonUp, wm.XButtonDown, wm.XButtonDblClk:

	}

	return false, 0
}

func registerApp(name string, style cs.ClassStyle) error {
	var once sync.Once
	once.Do(func() {
		fmt.Println("registering app")
		if name == "" {
			name = "GDL_app"
		}
		if style == 0 {
			style = cs.ByteAlignClient | cs.OwnDC | cs.VReDraw | cs.HReDraw
		}
		GDLAppClass = &w32.WindowClass{
			Name:  name,
			Style: style,
		}
		err := GDLAppClass.Register()
		if err != nil {
			panic(err) // if he application class cannot register, we are done.
		}

		// TODO: Port this:
		// GetModuleFileName(SDL_Instance, path, MAX_PATH);
		// ExtractIconEx(path, 0, &wcex.hIcon, &wcex.hIconSm, 1);
	})
	return nil
}