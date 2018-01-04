package wl

import (
	"github.com/elliotmr/gdl/wl/wlp"
	"sync"
	"fmt"
)

type Screen struct {
	output         *wlp.Output
	Mu             *sync.RWMutex
	X              int32
	Y              int32
	PhysicalWidth  int32
	PhysicalHeight int32
	Subpixel       int32
	Make           string
	Model          string
	Transform      int32
	Flags          uint32
	Width          int32
	Height         int32
	Refresh        int32
	Factor         int32
}

func (s *Screen) Geometry(x int32, y int32, physicalWidth int32, physicalHeight int32, subpixel int32, make string, model string, transform int32) {
	fmt.Printf("Geometry(%d, %d, %d, %d, %d, %s, %s, %d), called\n", x, y, physicalWidth, physicalHeight, subpixel, make, model, transform)
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.X = x
	s.Y = y
	s.PhysicalWidth = physicalWidth
	s.PhysicalHeight = physicalHeight
	s.Subpixel = subpixel
	s.Make = make
	s.Model = model
	s.Transform = transform
}

func (s *Screen) Mode(flags uint32, width int32, height int32, refresh int32) {
	fmt.Println("Mode called")
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Flags = flags
	s.Width = width
	s.Height = height
	s.Refresh = refresh
}

func (s *Screen) Done() {

}

func (s *Screen) Scale(factor int32) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Factor = factor
}
