package hook

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"unsafe"
)

const (
	relJmpSize  = 5  // E9 + rel32
	absJmpSize  = 14 // FF 25 00000000 + qword
	maxPrologue = 32 // bytes read from the target to relocate
	pageSize    = 0x1000
	allocGran   = 0x10000
	maxReach    = 0x7FFF0000 // stay inside +/- 2 GiB for rel32
)

type Detour struct {
	mtx        sync.Mutex
	target     uintptr
	trampoline uintptr
	page       uintptr
	saved      []byte
	installed  bool
}

func (d *Detour) Installed() bool {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	return d.installed
}

func (d *Detour) detourFunction(target, callback uintptr, prologue []byte) error {
	page, err := allocNear(target)
	if err != nil {
		return err
	}

	reloc, err := Relocate(prologue, target, page)
	if err != nil {
		return errors.Join(err, freePage(page))
	}

	trampLen := len(reloc.Code) + absJmpSize
	relayAddr := page + uintptr(trampLen)

	code := make([]byte, trampLen+absJmpSize)
	copy(code, reloc.Code)
	putAbsJmp(code[len(reloc.Code):], target+uintptr(reloc.SrcLen))
	putAbsJmp(code[trampLen:], callback)

	copy(unsafe.Slice((*byte)(unsafe.Pointer(page)), len(code)), code)

	d.target = target
	d.trampoline = page
	d.page = page
	d.saved = append([]byte(nil), prologue[:reloc.SrcLen]...)

	patch := make([]byte, reloc.SrcLen)
	for i := range patch {
		patch[i] = 0x90
	}

	putRelJmp(patch, target, relayAddr)

	err = writeProtected(target, patch)
	if err != nil {
		return errors.Join(err, freePage(page))
	}

	d.installed = true

	return nil
}

func (d *Detour) Restore() error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	if !d.installed {
		return nil
	}

	err := writeProtected(d.target, d.saved)

	d.trampoline = 0

	freeErr := freePage(d.page)

	d.installed = false

	return errors.Join(err, freeErr)
}

func (d *Detour) CallOriginal(args ...uintptr) uintptr {
	trampoline := d.trampoline
	if trampoline == 0 {
		return 0
	}

	return callNative(trampoline, args...)
}

func allocNear(target uintptr) (uintptr, error) {
	base := target &^ (allocGran - 1)

	for dist := uintptr(allocGran); dist < maxReach; dist += allocGran {
		for _, addr := range [2]uintptr{base - dist, base + dist} {
			p, err := allocPage(addr)
			if err == nil {
				return p, nil
			}
		}
	}

	return 0, fmt.Errorf("no free page within reach of %#x", target)
}

func writeProtected(addr uintptr, code []byte) error {
	prot, err := protect(addr, uintptr(len(code)), protectExecuteReadWrite)
	if err != nil {
		return fmt.Errorf("%s %#x: %w", protectName, addr, err)
	}

	copy(unsafe.Slice((*byte)(unsafe.Pointer(addr)), len(code)), code)

	_, err = protect(addr, uintptr(len(code)), prot)
	if err != nil {
		return fmt.Errorf("%s restore %#x: %w", protectName, addr, err)
	}

	return nil
}

func putAbsJmp(dst []byte, target uintptr) {
	dst[0] = 0xFF
	dst[1] = 0x25
	binary.LittleEndian.PutUint32(dst[2:], 0)
	binary.LittleEndian.PutUint64(dst[6:], uint64(target))
}

func putRelJmp(dst []byte, from, to uintptr) {
	dst[0] = 0xE9
	binary.LittleEndian.PutUint32(dst[1:], uint32(int32(int64(to)-int64(from)-relJmpSize)))
}
