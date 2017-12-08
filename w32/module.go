package w32

import (
	// #include <wtypes.h>
	// #include <winable.h>
	"C"
	"syscall"
	"unsafe"
	"github.com/pkg/errors"
)

var (
	moduser32 = syscall.NewLazyDLL("user32.dll")
	modkernel32 = syscall.NewLazyDLL("kernel32.dll")
	modopengl32 = syscall.NewLazyDLL("opengl32.dll")
)

var (
	procGetModuleHandle    = modkernel32.NewProc("GetModuleHandleW")
)

var inst *Module

func init() {
	var err error
	inst, err = GetModule("")
	if err != nil {
		panic(err)
	}
}

type Module struct {
	h syscall.Handle
}

func (m *Module) handle() syscall.Handle {
	if m == nil {
		return 0
	}
	return m.h
}

func GetModule(name string) (*Module, error) {
	var mn uintptr
	if name == "" {
		mn = 0
	} else {
		n, err := syscall.UTF16PtrFromString(name)
		if err != nil {
			errors.Wrap(err, "invalid module name")
		}
		mn = uintptr(unsafe.Pointer(n))
	}
	ret, _, err := procGetModuleHandle.Call(mn)
	if ret == 0 {
		if err.(syscall.Errno) != 0 {
			return nil, errors.Wrap(err, "error calling user32")
		} else {
			return nil, syscall.EINVAL
		}
	}
	return &Module{h: syscall.Handle(ret)}, nil
}

