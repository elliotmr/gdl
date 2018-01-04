package wlp

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/pkg/errors"
)

type constructor func(*Context) Object

var constructors map[string]constructor

type Object interface {
	ID() uint32
	Type() string
	dispatch(opCode uint16, payload []byte, file *os.File)
	setListener(listener interface{}) error
}

type global struct {
	Name      uint32
	Interface string
	Version   uint32
}

// NewContext will create a new "global" context for a wayland connection
func NewContext(conn *net.UnixConn) *Context {
	return &Context{
		mu:          &sync.Mutex{},
		c:           conn,
		buf:         &bytes.Buffer{},
		obj:         make(map[uint32]Object),
		glb:         make(map[uint32]global),
		glbByString: make(map[string][]global),
	}
}

type Context struct {
	*Display
	*Registry

	mu          *sync.Mutex
	c           *net.UnixConn
	buf         *bytes.Buffer
	obj         map[uint32]Object
	glb         map[uint32]global
	glbByString map[string][]global
	last        uint32
	Err         error
}

func (c *Context) decodeFD(n int, oob []byte) (*os.File, error) {
	if n == 0 {
		return nil, nil
	}
	scms, err := syscall.ParseSocketControlMessage(oob[:n])
	if err != nil {
		return nil, errors.Wrap(err, "ParseSocketControlMessage failed")
	}
	if len(scms) != 1 {
		return nil, errors.Errorf("SocketControlMessage count not 1: %v\n", len(scms))
	}
	scm := scms[0]
	fds, err := syscall.ParseUnixRights(&scm)
	if err != nil {
		return nil, errors.Wrapf(err, "ParseUnixRights failed")
	}
	if len(fds) != 1 {
		return nil, errors.Errorf("recvfd: fd count not 1: %v", len(fds))
	}
	return os.NewFile(uintptr(fds[0]), "wayland-fd"), nil
}

func (c *Context) encodeFD(f *os.File) []byte {
	if f == nil {
		return nil
	}
	return syscall.UnixRights(int(f.Fd()))
}

func (c *Context) readLoop() {
	buf := make([]byte, 65535)
	oobBuf := make([]byte, os.Getpagesize())
	j := 0
outer:
	for {
		n, oobn, _, _, err := c.c.ReadMsgUnix(buf[j:], oobBuf)
		n += j
		if err != nil {
			fmt.Printf("readloop error: %v\n", err)
			return
		}
		file, err := c.decodeFD(oobn, oobBuf)
		if err != nil {
			fmt.Printf("readloop error: %v\n", err)
			return
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
			id, opcode, size := DecodeHeader(buf[i:])
			payload := make([]byte, size-8)
			m := copy(payload, buf[i+8:])
			if m < size-8 {
				j = copy(buf, buf[n-i:n])
				continue outer
			}
			i += m + 8
			fmt.Printf("Event: %d, %d, %d, %v\n", id, size, opcode, payload)
			c.obj[id].dispatch(opcode, payload, file)
			if c.Err != nil {
				// trigger pending callbacks
				for _, obj := range c.obj {
					cb, ok := obj.(*Callback)
					if !ok {
						continue
					}
					cb.dispatch(opCodeCallbackDone, []byte{0, 0, 0, 0}, nil)
				}
				return
			}
		}
	}
}

func (c *Context) Start() {
	c.Display = newDisplay(c).(*Display)
	c.Display.setListener(c)
	go c.readLoop()
	c.Registry, _ = c.Display.GetRegistry(c)
}

func (c *Context) next() uint32 {
	return atomic.AddUint32(&c.last, 1)
}

// Error handles stores global errors from the
func (c *Context) Error(objectID uint32, code uint32, message string) {
	c.Err = errors.Errorf("obj: %d, code: %d -> %s", objectID, code, message)
}

func (c *Context) DeleteID(id uint32) {
	delete(c.obj, id)
	return
}

// Global is an implementation of the RegistryListener interface. It will receive
// callbacks from the Global registry and stores them in a registry map.
func (c *Context) Global(name uint32, iface string, version uint32) {
	glb := global{
		Name:      name,
		Interface: iface,
		Version:   version,
	}
	c.glb[name] = glb
	c.glbByString[iface] = append(
		c.glbByString[iface],
		glb,
	)
	fmt.Println("Added global: ", iface)
}

// GlobalRemove is an implementation of the RegistryListener interface for removing
// global objects from the client when they are no longer present in the server.
func (c *Context) GlobalRemove(name uint32) {
	glb, exists := c.glb[name]
	if !exists {
		return
	}
	delete(c.glb, name)
	if len(c.glbByString[glb.Interface]) == 0 {
		return
	}
	b := c.glbByString[glb.Interface][:0]
	for _, g := range c.glbByString[glb.Interface] {
		if g.Name != name {
			b = append(b, g)
		}
	}
}

func (c *Context) BindGlobalIndex(ifname string, listener interface{}, i int) (Object, error) {
	if i > len(c.glbByString[ifname]) {
		return nil, errors.Errorf("index: %d out of range for interface: %s", i, ifname)
	}
	glb := c.glbByString[ifname][i]
	o := constructors[ifname](c)
	err := o.setListener(listener)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid listener")
	}
	err = c.Bind(glb.Name, glb.Interface, glb.Version, o.ID())
	if err != nil {
		return nil, errors.Wrapf(err, "unable to bind object: %s", glb.Interface)
	}
	return o, nil
}

func (c *Context) NumGlobals(ifname string) int {
	return len(c.glbByString[ifname])
}

func (c *Context) BindGlobal(ifname string, listener interface{}) (Object, error) {
	if len(c.glbByString[ifname]) != 1 {
		return nil, errors.New("BidGlobal requires exactly one instance")
	}
	return c.BindGlobalIndex(ifname, listener, 0)
}
