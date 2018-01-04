package wl

import (
	"github.com/elliotmr/gdl/wl/wlp"
	"fmt"
	"github.com/pkg/errors"
)

type surfaceCb struct {
	w *Window
}

func (scb *surfaceCb) Configure(serial uint32) {
	fmt.Printf("Configure(serial: %X) -> ACK\n", serial)
	scb.w.AckConfigure(serial)
}

type buffer struct {
	*wlp.Buffer

	bound bool
	w     *Window
}

func (b *buffer) Release() {
	b.bound = false
}

type Window struct {
	*wlp.Surface
	*wlp.ZxdgSurfaceV6
	*wlp.ZxdgToplevelV6

	c *Client

	width  int32
	height int32

	pool          *wlp.ShmPool
	data          []byte
	buffers       [2]buffer
	currentBuffer int
	pageSize      int32

	scb *surfaceCb
}

func (w *Window) InitGraphics() error {
	var err error
	w.pageSize = 0
	for _, scr := range w.c.Screens {
		scr.Mu.RLock()
		scrMax := scr.Width * scr.Height * 4
		if scrMax > w.pageSize {
			w.pageSize = scrMax
		}
	}
	w.pool, w.data, err = w.c.CreateMemoryPool(uint32(w.pageSize) * 2)
	if err != nil {
		return errors.Wrap(err, "unable to create memory pool")
	}

	return w.c.Roundtrip()
}

func (w *Window) Release() {
	panic("implement me")
}

func (w *Window) Configure(width int32, height int32, states []byte) {
	fmt.Printf("Configuring Window (%d, %d, %v)\n", width, height, states)
	if w.buffers[0].bound {
		w.buffers[0].Destroy()
	}
	if w.buffers[1].bound {
		w.buffers[1].Destroy()
	}
	if w.buffers[0].bound || w.buffers[1].bound {
		w.c.Roundtrip()
	}

	if w.currentBuffer == 1 {
		w.buffers[0].Buffer, _ = w.pool.CreateBuffer(&w.buffers[0], 0, width, height, width, wlp.ShmFormatXrgb8888)
		w.buffers[0].bound = true
		w.Attach(w.buffers[0].Buffer.ID(), 0, 0)
		w.currentBuffer = 0
	} else {
		w.buffers[1].Buffer, _ = w.pool.CreateBuffer(&w.buffers[0], w.pageSize, width, height, width, wlp.ShmFormatXrgb8888)
		w.buffers[1].bound = true
		w.Attach(w.buffers[1].Buffer.ID(), 0, 0)
		w.currentBuffer = 1
	}
	w.Commit()
}

func (w *Window) Close() {
	panic("implement me")
}

func (w *Window) Enter(output uint32) {
	fmt.Println("entering window")
}

func (w *Window) Leave(output uint32) {
	fmt.Println("leaving window")
}
