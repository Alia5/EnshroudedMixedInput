package hook

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// MSVC leaves runtime type information in every binary with virtual classes.
// Three read-only records link a class name to its method table, and each one
// can be found by scanning for a value the previous one gave us:
//
//	type descriptor    +0x10  the mangled class name, e.g. ".?AVMyClass@@"
//	object locator     +0x00  signature, 1 on x64
//	                   +0x0c  image offset of the type descriptor
//	method table       [-1]   pointer to the object locator
//	                   [0..]  the function pointers
const (
	typeDescriptorName    = 0x10
	locatorSignature      = 1
	locatorTypeDescriptor = 0x0c

	jumpRel32 = 0xe9
)

var (
	thunkAdjustByte = uintptr(len(thunk))
	thunkJumpOpcode = thunkAdjustByte + 1
	thunkJumpRel    = thunkJumpOpcode + 1
	thunkSize       = thunkJumpRel + unsafe.Sizeof(int32(0))
)

// thunk is `add rcx, imm8; jmp rel32`: the adapter MSVC emits when a method
// table entry forwards to a method of an embedded object. Bytes: the add's
// opcode, then imm8, then 0xe9 and rel32.
var thunk = []byte{0x48, 0x83, 0xc1}

func Image(module uintptr) ([]byte, error) {
	var info windows.ModuleInfo

	err := windows.GetModuleInformation(windows.CurrentProcess(), windows.Handle(module), &info, uint32(unsafe.Sizeof(info)))
	if err != nil {
		return nil, fmt.Errorf("GetModuleInformation %#x: %w", module, err)
	}

	return unsafe.Slice((*byte)(unsafe.Pointer(module)), info.SizeOfImage), nil
}

func MethodTable(module uintptr, typeName string) (uintptr, error) {
	image, err := Image(module)
	if err != nil {
		return 0, err
	}

	name := bytes.Index(image, append([]byte(typeName), 0))
	if name < 0 {
		return 0, fmt.Errorf("%w: %s", ErrTypeNameNotFound, typeName)
	}

	descriptor := uint32(name - typeDescriptorName)

	locator := -1

	for pos := 0; locator < 0; {
		i := bytes.Index(image[pos:], binary.LittleEndian.AppendUint32(nil, descriptor))
		if i < 0 {
			break
		}

		candidate := pos + i - locatorTypeDescriptor

		if candidate >= 0 && binary.LittleEndian.Uint32(image[candidate:]) == locatorSignature {
			locator = candidate
		}

		pos += i + 1
	}

	if locator < 0 {
		return 0, fmt.Errorf("%w: %s", ErrObjectLocatorNotFound, typeName)
	}

	ref := bytes.Index(image, binary.LittleEndian.AppendUint64(nil, uint64(module)+uint64(locator)))
	if ref < 0 {
		return 0, fmt.Errorf("%w: %s", ErrMethodTableNotFound, typeName)
	}

	return module + uintptr(ref) + unsafe.Sizeof(uintptr(0)), nil
}

func FollowThunk(module uintptr, image []byte, addr uintptr) (target, adjust uintptr, ok bool) {
	off := addr - module
	if off+thunkSize > uintptr(len(image)) || !bytes.HasPrefix(image[off:], thunk) || image[off+thunkJumpOpcode] != jumpRel32 {
		return 0, 0, false
	}

	rel := int32(binary.LittleEndian.Uint32(image[off+thunkJumpRel:]))

	return addr + thunkSize + uintptr(rel), uintptr(image[off+thunkAdjustByte]), true
}

func sectionEnd(image []byte, rva uintptr) uintptr {
	file, err := pe.NewFile(bytes.NewReader(image))
	if err != nil {
		return uintptr(len(image))
	}
	defer file.Close() //nolint:errcheck

	for _, section := range file.Sections {
		address := uintptr(section.VirtualAddress)
		size := uintptr(section.VirtualSize)

		if rva >= address && rva < address+size {
			return min(address+size, uintptr(len(image)))
		}
	}

	return uintptr(len(image))
}
