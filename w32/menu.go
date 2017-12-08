package w32

import "syscall"

type Menu struct {
	h syscall.Handle
}

func (m *Menu) handle() syscall.Handle {
	if m == nil {
		return 0
	}
	return m.h
}