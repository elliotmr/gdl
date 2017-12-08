package w32

import "syscall"

type Brush struct {
	h syscall.Handle
}

func (b *Brush) handle() syscall.Handle {
	if b == nil {
		return 0
	}
	return b.h
}