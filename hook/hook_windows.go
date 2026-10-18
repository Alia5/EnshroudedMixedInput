package hook

import (
	"bytes"
	debugpe "debug/pe"
	"encoding/binary"
	"fmt"
	"iter"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	protectName             = "VirtualProtect"
	protectReadWrite        = windows.PAGE_READWRITE
	protectExecuteReadWrite = windows.PAGE_EXECUTE_READWRITE
)

const runtimeFunctionSize = 12

const unwindFlagChainInfo = 0x4

const (
	dosSignature    = 0x5a4d
	peSignature     = 0x00004550
	pe64Magic       = 0x20b
	lfanewOffset    = 0x3c
	optionalHeader  = 24
	importDirectory = optionalHeader + 112 + 8

	importDescriptorSize    = 20
	importDescriptorNames   = 0
	importDescriptorDLLName = 12
	importDescriptorSlots   = 16
	importThunkSize         = 8
	importOrdinalFlag       = 1 << 63
	importHintSize          = 2
)

func Module(name string) (uintptr, error) {
	var h windows.Handle

	err := windows.GetModuleHandleEx(0, windows.StringToUTF16Ptr(name), &h)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}

	return uintptr(h), nil
}

func protect(addr, size uintptr, protection uint32) (uint32, error) {
	var prot uint32

	err := windows.VirtualProtect(addr, size, protection, &prot)

	return prot, err
}

// ImportEntry returns the IAT slot of dll!function in an already mapped module.
func ImportEntry(module uintptr, dll, function string) (uintptr, error) {
	if *(*uint16)(unsafe.Pointer(module)) != dosSignature {
		return 0, fmt.Errorf("%#x: %w", module, ErrNotSupported)
	}

	headers := module + uintptr(*(*uint32)(unsafe.Pointer(module + lfanewOffset)))

	if *(*uint32)(unsafe.Pointer(headers)) != peSignature ||
		*(*uint16)(unsafe.Pointer(headers + optionalHeader)) != pe64Magic {
		return 0, fmt.Errorf("%#x: %w", module, ErrNotSupported)
	}

	descriptor := module + uintptr(*(*uint32)(unsafe.Pointer(headers + importDirectory)))

	for ; ; descriptor += importDescriptorSize {
		names := *(*uint32)(unsafe.Pointer(descriptor + importDescriptorNames))
		imported := *(*uint32)(unsafe.Pointer(descriptor + importDescriptorDLLName))
		slots := *(*uint32)(unsafe.Pointer(descriptor + importDescriptorSlots))

		if imported == 0 && slots == 0 {
			break
		}

		if !strings.EqualFold(windows.BytePtrToString((*byte)(unsafe.Pointer(module+uintptr(imported)))), dll) {
			continue
		}

		if names == 0 {
			names = slots
		}

		for i := uintptr(0); ; i++ {
			name := *(*uint64)(unsafe.Pointer(module + uintptr(names) + i*importThunkSize))
			if name == 0 {
				break
			}

			if name&importOrdinalFlag != 0 {
				continue
			}

			if windows.BytePtrToString((*byte)(unsafe.Pointer(module+uintptr(name)+importHintSize))) == function {
				return module + uintptr(slots) + i*importThunkSize, nil
			}
		}
	}

	return 0, fmt.Errorf("%s!%s not imported by %#x", dll, function, module)
}

func FunctionStart(module uintptr, image []byte, address uintptr) (uintptr, error) {
	file, err := debugpe.NewFile(bytes.NewReader(image))
	if err != nil {
		return 0, err
	}
	defer file.Close() //nolint:errcheck

	header, ok := file.OptionalHeader.(*debugpe.OptionalHeader64)
	if !ok {
		return 0, fmt.Errorf("%w: not a 64-bit image", ErrFunctionNotFound)
	}

	directory := header.DataDirectory[debugpe.IMAGE_DIRECTORY_ENTRY_EXCEPTION]
	rva := uint32(address - module)

	for entry := uintptr(directory.VirtualAddress); entry+runtimeFunctionSize <= uintptr(directory.VirtualAddress+directory.Size); entry += runtimeFunctionSize {
		begin, end, unwind := runtimeFunction(image, entry)

		if rva < begin || rva >= end {
			continue
		}

		for image[unwind]>>3&unwindFlagChainInfo != 0 {
			codes := uintptr(image[unwind+2])
			chained := uintptr(unwind) + 4 + ((codes+1)&^1)*2
			begin, _, unwind = runtimeFunction(image, chained)
		}

		return module + uintptr(begin), nil
	}

	return 0, fmt.Errorf("%w: %#x", ErrFunctionNotFound, address)
}

func Functions(image []byte) iter.Seq2[uintptr, uintptr] {
	return func(yield func(begin, end uintptr) bool) {
		file, err := debugpe.NewFile(bytes.NewReader(image))
		if err != nil {
			return
		}
		defer file.Close() //nolint:errcheck

		header, ok := file.OptionalHeader.(*debugpe.OptionalHeader64)
		if !ok {
			return
		}

		directory := header.DataDirectory[debugpe.IMAGE_DIRECTORY_ENTRY_EXCEPTION]
		last := uintptr(directory.VirtualAddress + directory.Size)

		for entry := uintptr(directory.VirtualAddress); entry+runtimeFunctionSize <= last; entry += runtimeFunctionSize {
			begin, end, _ := runtimeFunction(image, entry)

			if !yield(uintptr(begin), uintptr(end)) {
				return
			}
		}
	}
}

func MappedRanges(image []byte) iter.Seq2[uintptr, uintptr] {
	return func(yield func(begin, end uintptr) bool) {
		yield(0, uintptr(len(image)))
	}
}

func callNative(fn uintptr, args ...uintptr) uintptr {
	ret, _, _ := syscall.SyscallN(fn, args...)

	return ret
}

func runtimeFunction(image []byte, entry uintptr) (begin, end, unwind uint32) {
	return binary.LittleEndian.Uint32(image[entry:]),
		binary.LittleEndian.Uint32(image[entry+4:]),
		binary.LittleEndian.Uint32(image[entry+8:])
}
