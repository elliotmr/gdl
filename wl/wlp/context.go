package wlp

import (
	"net"
	"sync"
	"bytes"
	"sync/atomic"
	"os"
	"github.com/pkg/errors"
)

type object interface {
	id() uint32
	dispatch(opCode uint16, payload []byte, file *os.File)
}

type context struct {
	mu   *sync.Mutex
	c    *net.UnixConn
	buf  *bytes.Buffer
	obj  map[uint32]object
	last uint32
	err  error
}

func (c *context) next() uint32 {
	return atomic.AddUint32(&c.last, 1)
}

func (c *context) Error(objectID uint32, code uint32, message string) {
	c.err = errors.Errorf("obj: %d, code: %d -> %s", objectID, code, message)
}

func (c *context) DeleteID(id uint32) {
	delete(c.obj, id)
	return
}
