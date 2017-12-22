package wl

import (
	"net"
	"os"
	"github.com/pkg/errors"
	"sync"
	"sync/atomic"
	"path/filepath"
	"fmt"
	"encoding/hex"
	//"io"
	"syscall"
	"bytes"
	"io/ioutil"
)

var atomicIDCounter uint32

func GetNewID() ObjectID {
	return ObjectID(atomic.AddUint32(&atomicIDCounter, 1))
}

type ObjectID uint32

type Object interface {
	ID() ObjectID
	Dispatch(opCode uint16, payload []byte, file *os.File)
}

type Global struct {
	Name      uint32
	Interface string
	Version   uint32
}

type Client struct {
	*Display

	registry   *Registry
	compositor *Compositor
	shm        *Shm
	shell      *Shell

	conn    *net.UnixConn
	mu      *sync.Mutex
	buf     *bytes.Buffer
	objects map[ObjectID]Object
	globals map[uint32]Global
	err     error
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

func (c *Client) readLoop() {
	buf := make([]byte, 65535)
	oobBuf := make([]byte, os.Getpagesize())
	j := 0
outer:
	for {
		var file *os.File
		n, oobn, _, _, err := c.conn.ReadMsgUnix(buf[j:], oobBuf)
		n += j
		if err != nil {
			fmt.Printf("readloop error: %v\n", err)
			return
		}
		if oobn != 0 {
			scms, err := syscall.ParseSocketControlMessage(oobBuf[:oobn])
			if err != nil {
				fmt.Printf("ParseSocketControlMessage failed: %v\n", err)
				return
			}
			if len(scms) != 1 {
				fmt.Printf("SocketControlMessage count not 1: %v\n", len(scms))
				return
			}
			scm := scms[0]
			fds, err := syscall.ParseUnixRights(&scm)
			if err != nil {
				fmt.Errorf("ParseUnixRights failed %v\n", err)
				return
			}
			if len(fds) != 1 {
				fmt.Errorf("recvfd: fd count not 1: %v", len(fds))
				return
			}
			file = os.NewFile(uintptr(fds[0]), "wayland-fd")
		}
		// TODO: remove after debugging
		fmt.Println("Received:")
		fmt.Println(hex.Dump(buf[:n]))

		i := 0
		for i < n {
			if n-i < 8 {
				j = copy(buf, buf[n-i:n])
				continue outer
			}
			oid := ObjectID(hostByteOrder.Uint32(buf[i:i+4]))
			arg2 := hostByteOrder.Uint32(buf[i+4:i+8])
			opCode := uint16(arg2 & 0xFF)
			size := int(arg2 >> 16)
			payload := make([]byte, size-8)
			m := copy(payload, buf[i+8:])
			if m < size-8 {
				j = copy(buf, buf[n-i:n])
				continue outer
			}
			i += m + 8
			fmt.Printf("Event: %d, %d, %d, %v\n", oid, size, opCode, payload)
			c.objects[oid].Dispatch(opCode, payload, file)
		}
	}
}

func (c *Client) Connect(sockName string) error {
	// TODO(mde): Add support for connecting to an open file descriptor
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
	c.conn, err = net.DialUnix("unix", nil, addr)
	if err != nil {
		return errors.Wrapf(err, "unable to connect to wayland server at (%s)", absSockName)
	}
	c.buf = &bytes.Buffer{}
	c.mu = &sync.Mutex{}
	c.objects = make(map[ObjectID]Object)
	c.globals = make(map[uint32]Global)
	c.Display = c.NewDisplay()
	c.SetListener(c)
	go c.readLoop()
	c.registry, c.err = c.Display.GetRegistry()
	if c.err != nil {
		return c.err
	}
	c.registry.SetListener(c)
	c.err = c.Roundtrip()

	return c.err
}

// Roundtrip is a convenience wrapper around Sync. It will sleep the calling
// go-routine until all pending wayland commands are processed.
func (c *Client) Roundtrip() error {
	cb, err := c.Sync()
	if err != nil {
		return errors.Wrap(err, "unable to create display sync")
	}
	cbl := NewCallbackListener()
	cbl.L.Lock()
	cb.SetListener(cbl)
	cbl.Wait()
	return c.err
}

// Error implements a display listener callback method, it will be called
// if there is a global error for the connection. The callback method will save
// the error internally to the client, so future calls will fail.
func (c *Client) Error(objectID uint32, code uint32, message string) {
	c.Disconnect()
	c.err = errors.Errorf("obj: %d, code: %d -> %s", objectID, code, message)
}

// DeleteID implements the display listener callback method. It will remove an object
// from the
func (c *Client) DeleteID(id uint32) {
	delete(c.objects, ObjectID(id))
	return
}

// Implement Registry Listener
func (c *Client) Global(name uint32, iface string, version uint32) {
	c.globals[name] = Global{
		Name:      name,
		Interface: iface,
		Version:   version,
	}
	switch iface {
	case "wl_compositor":
		c.compositor = c.NewCompositor()
		c.registry.Bind(name, c.compositor.ID())
	case "wl_shm":
		c.shm = c.NewShm()
		c.registry.Bind(name, c.shm.ID())
	case "wl_shell":
		c.shell = c.NewShell()
		c.registry.Bind(name, c.shell.ID())
	}
}

func (c *Client) GlobalRemove(name uint32) {
	delete(c.globals, name)
}

func (c *Client) CreateMemoryPool(size uint32) (*ShmPool, error) {
	f, err := ioutil.TempFile(os.Getenv("XDG_RUNTIME_DIR"), "shm")
	if err != nil {
		return nil, errors.Wrap(err, "unable to creat backinf file")
	}
	// leave unlink out for debugging purposes
	// syscall.Unlink(f.Name())
	syscall.Mmap()
}

func (c *Client) Disconnect() error {
	return c.conn.Close()
}
