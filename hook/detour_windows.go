package hook

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (d *Detour) Apply(target, callback uintptr) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if d.installed {
		return fmt.Errorf("detour already installed at %#x", d.target)
	}

	prologue := make([]byte, maxPrologue)
	copy(prologue, unsafe.Slice((*byte)(unsafe.Pointer(target)), maxPrologue))

	return d.detourFunction(target, callback, prologue)
}

func allocPage(addr uintptr) (uintptr, error) {
	return windows.VirtualAlloc(
		addr, pageSize,
		windows.MEM_RESERVE|windows.MEM_COMMIT,
		windows.PAGE_EXECUTE_READWRITE,
	)
}

func freePage(page uintptr) error {
	return windows.VirtualFree(page, 0, windows.MEM_RELEASE)
}
