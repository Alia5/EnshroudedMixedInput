package sigtest

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"unsafe"
)

const (
	relocBlockHeaderSize = 8
	relocTypeShift       = 12
	relocOffsetMask      = 0x0fff
	relocAbsolute        = 0  // IMAGE_REL_BASED_ABSOLUTE, padding
	relocDir64           = 10 // IMAGE_REL_BASED_DIR64
)

var ErrNotPE64 = errors.New("not a 64-bit PE image")

// MapImage lays a PE file out at its section RVAs in Go memory and applies
// base relocations, so the result reads like the loader-mapped module would.
// The image is never executed.
func MapImage(path string) (uintptr, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, nil, err
	}

	file, err := pe.NewFile(bytes.NewReader(raw))
	if err != nil {
		return 0, nil, err
	}
	defer file.Close() //nolint:errcheck

	header, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok {
		return 0, nil, ErrNotPE64
	}

	image := make([]byte, header.SizeOfImage)
	copy(image, raw[:header.SizeOfHeaders])

	for _, section := range file.Sections {
		size := min(section.Size, section.VirtualSize)
		if size == 0 {
			continue
		}

		copy(image[section.VirtualAddress:section.VirtualAddress+size], raw[section.Offset:section.Offset+size])
	}

	module := uintptr(unsafe.Pointer(&image[0]))

	err = relocate(image, header, uint64(module)-header.ImageBase)
	if err != nil {
		return 0, nil, fmt.Errorf("relocate: %w", err)
	}

	return module, image, nil
}

func relocate(image []byte, header *pe.OptionalHeader64, delta uint64) error {
	directory := header.DataDirectory[pe.IMAGE_DIRECTORY_ENTRY_BASERELOC]

	for block := directory.VirtualAddress; block+relocBlockHeaderSize <= directory.VirtualAddress+directory.Size; {
		page := binary.LittleEndian.Uint32(image[block:])
		size := binary.LittleEndian.Uint32(image[block+4:])

		if size < relocBlockHeaderSize {
			return fmt.Errorf("block %#x: size %#x", block, size)
		}

		for entry := block + relocBlockHeaderSize; entry+2 <= block+size; entry += 2 {
			value := binary.LittleEndian.Uint16(image[entry:])

			switch value >> relocTypeShift {
			case relocAbsolute:
			case relocDir64:
				at := page + uint32(value&relocOffsetMask)
				binary.LittleEndian.PutUint64(image[at:], binary.LittleEndian.Uint64(image[at:])+delta)
			default:
				return fmt.Errorf("block %#x: relocation type %d", block, value>>relocTypeShift)
			}
		}

		block += size
	}

	return nil
}
