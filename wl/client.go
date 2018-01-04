package wl

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/elliotmr/gdl/wl/wlp"
	"github.com/pkg/errors"
	"fmt"
)


type Client struct {
	ctx        *wlp.Context
	compositor *wlp.Compositor
	shm        *wlp.Shm
	shell      *wlp.ZxdgShellV6
	Screens    []*Screen
	Windows    []*Window
}

type cbListener struct {
	*sync.Cond
	Data uint32
}

func NewCallbackListener() *cbListener {
	return &cbListener{Cond: sync.NewCond(&sync.Mutex{})}
}

func (cbl *cbListener) Done(callbackData uint32) {
	cbl.Data = callbackData
	cbl.Broadcast()
	return
}

// TODO: multiple fds

// Implements ZydgShellV6Listener
func (c *Client) Ping(serial uint32) {
	c.shell.Pong(serial)
}

// Implements Shm Listener
func (c *Client) Format(format uint32) {
	fmt.Println("Valid Format: ", format)
}


func (c *Client) Connect(sockName string) error {
	// TODO: Add support for connecting to an open file descriptor
	if sockName == "" {
		sockName = os.Getenv("WAYLAND_DISPLAY")
	}
	if sockName == "" {
		sockName = "wayland-0"
	}

	pathIsAbsolute := sockName[0] == '/'
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if !pathIsAbsolute && runtimeDir == "" {
		return errors.New("XDG_RUNTIME_DIR is not set in environment")
	}

	absSockName := filepath.Join(runtimeDir, sockName)
	addr, err := net.ResolveUnixAddr("unix", absSockName)
	if err != nil {
		return errors.Wrapf(err, "unable to resolve unix socket address (%s)", absSockName)
	}
	conn, err := net.DialUnix("unix", nil, addr)
	if err != nil {
		return errors.Wrapf(err, "unable to connect to wayland server at (%s)", absSockName)
	}
	c.ctx = wlp.NewContext(conn)
	c.ctx.Start()
	err = c.Roundtrip()
	if err != nil {
		return errors.Wrap(err, "starting context failed")
	}

	cmp, err := c.ctx.BindGlobal("wl_compositor", c)
	if err != nil {
		return errors.Wrap(err, "unable to bind wl_compositor")
	}
	c.compositor = cmp.(*wlp.Compositor)


	shm, err := c.ctx.BindGlobal("wl_shm", c)
	if err != nil {
		return errors.Wrap(err, "unable to bind wl_shm")
	}
	c.shm = shm.(*wlp.Shm)

	for i := 0; i < c.ctx.NumGlobals("wl_output"); i++ {
		scr := &Screen{
			Mu: &sync.RWMutex{},
			Factor: 1,
		}
		output, err := c.ctx.BindGlobalIndex("wl_output", scr, i)
		if err != nil {
			return errors.Wrap(err, "unable to bind wl_output")
		}
		scr.output = output.(*wlp.Output)
		c.Screens = append(c.Screens, scr)
	}

	shell, err := c.ctx.BindGlobal("zxdg_shell_v6", c)
	if err != nil {
		return errors.Wrap(err, "unable to bind zxdg_shell_v6")
	}
	c.shell = shell.(*wlp.ZxdgShellV6)

	return c.Roundtrip()
}

// Roundtrip is a convenience wrapper around Sync. It will sleep the calling
// go-routine until all pending wayland commands are processed.
func (c *Client) Roundtrip() error {
	cbl := NewCallbackListener()
	cbl.L.Lock()
	_, err := c.ctx.Display.Sync(cbl)
	if err != nil {
		return errors.Wrap(err, "unable to create display sync")
	}
	cbl.Wait()
	return c.ctx.Err
}

func (c *Client) CreateWindow() (*Window, error) {
	var err error

	w := &Window{c: c}
	w.buffers[0].w = w
	w.buffers[1].w = w
	w.scb = &surfaceCb{w: w}


	w.Surface, err = c.compositor.CreateSurface(w)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create surface")
	}
	w.ZxdgSurfaceV6, err = c.shell.GetXdgSurface(w.scb, w.Surface.ID())
	if err != nil {
		return nil, errors.Wrap(err, "unable to create xdg_surface")
	}
	w.ZxdgToplevelV6, err = w.ZxdgSurfaceV6.GetToplevel(w)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create xdg_toplevel")
	}

	err = w.InitGraphics()
	if err != nil {
		return nil, errors.Wrap(err, "unable to init graphics on window")
	}
	return w, c.Roundtrip()
}

func (c *Client) CreateMemoryPool(size uint32) (*wlp.ShmPool, []byte, error) {
	f, err := ioutil.TempFile(os.Getenv("XDG_RUNTIME_DIR"), "shm")
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create backing file")
	}
	// leave unlink out for debugging purposes
	syscall.Unlink(f.Name())
	err = f.Truncate(int64(size))
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to resize backing file")
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to mmap temp file")
	}

	pool, err := c.shm.CreatePool(c, f, int32(size * 2))
	if err != nil {
		return nil, nil, errors.Wrap(err, "unable to create shm pool")
	}

	return pool, data, nil
}

