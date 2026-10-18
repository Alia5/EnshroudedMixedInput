package hook

import (
	"bytes"
	"fmt"
	"sync"
	"unsafe"
)

type CodePatch struct {
	mtx     sync.Mutex
	address uintptr
	saved   []byte
}

func (c *CodePatch) Apply(address uintptr, code []byte) error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.saved != nil {
		return fmt.Errorf("%w: %#x", ErrCodeAlreadyPatched, c.address)
	}

	saved := bytes.Clone(unsafe.Slice((*byte)(unsafe.Pointer(address)), len(code)))

	err := writeProtected(address, code)
	if err != nil {
		return err
	}

	c.address = address
	c.saved = saved

	return nil
}

func (c *CodePatch) Installed() bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	return c.saved != nil
}

func (c *CodePatch) Restore() error {
	c.mtx.Lock()
	defer c.mtx.Unlock()

	if c.saved == nil {
		return nil
	}

	err := writeProtected(c.address, c.saved)
	if err != nil {
		return err
	}

	c.saved = nil

	return nil
}
