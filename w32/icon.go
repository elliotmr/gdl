package w32

import "syscall"

type Icon struct {
	h syscall.Handle
}

func (i *Icon) handle() syscall.Handle {
	if i == nil {
		return 0
	}
	return i.h
}