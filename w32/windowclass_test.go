package w32

import (
	"fmt"
	"sync"
	"testing"

	"github.com/elliotmr/gdl/w32/types/cs"
	//"github.com/elliotmr/gdl/w32/types/wm"
	"github.com/elliotmr/gdl/w32/types/ws"
	"github.com/stretchr/testify/assert"
)

type TestWindowHandler struct {
	t  *testing.T
	wg *sync.WaitGroup
}

func (twh *TestWindowHandler) OnMessage(uMsg uint32, wParam, lParam uintptr) (bool, uintptr) {
	fmt.Printf("Received Message: 0x%04X\n", uMsg)
	return false, 0
}

func TestWindowClassRegister(t *testing.T) {
	wc := &WindowClass{
		Name:  "TestWindowClass",
		Style: cs.VReDraw | cs.HReDraw,
		Background: &Brush{h: 6},
	}
	assert.NoError(t, wc.Register())
	props := WindowProps{
		Name:   "TestWindow",
		Style:  ws.OverlappedWindow | ws.Visible,
		X:      WPUseDefault,
		Y:      WPUseDefault,
		Height: WPUseDefault,
		Width:  WPUseDefault,
	}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	handler := &TestWindowHandler{t: t, wg: wg}
	w, err := wc.New(handler, props)
	assert.NoError(t, err)
	go func() {
		err = w.Run()
		if err != nil {
			t.Error(err)
		}
		wg.Done()
	}()
	wg.Wait()
	assert.NoError(t, wc.UnRegister())
}
