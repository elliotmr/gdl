package w32

import "syscall"

type Cursor struct {
	h syscall.Handle
}

func (c *Cursor) handle() syscall.Handle {
	if c == nil {
		return 0
	}
	return c.h
}