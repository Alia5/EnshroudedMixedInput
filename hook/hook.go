package hook

import (
	"errors"
	"fmt"
	"log/slog"
	"unsafe"

	"golang.org/x/arch/x86/x86asm"

	"github.com/Alia5/EnshroudedMixedInput/pattern"
)

var (
	ErrNotSupported      = errors.New("not supported on this platform yet")
	ErrSlotNotFound      = errors.New("method table slot not found")
	ErrPatternNotFound   = errors.New("pattern not found")
	ErrAmbiguousMatch    = errors.New("pattern matched at more than one address")
	ErrFunctionNotFound  = errors.New("no exception table entry contains the address")
	ErrReferenceNotFound = errors.New("no instruction references the address")

	ErrCodeAlreadyPatched = errors.New("code patch already installed")

	ErrTypeNameNotFound      = errors.New("rtti type name not found")
	ErrObjectLocatorNotFound = errors.New("rtti object locator not found")
	ErrMethodTableNotFound   = errors.New("rtti method table not found")
)

func RecoverPanic(f func()) {
	defer func() {
		panic := recover()
		if panic != nil {
			slog.Error("recovered from panic", "panic", panic)
		}
	}()

	f()
}

func Swap(addr, value uintptr) (uintptr, error) {
	slot := (*uintptr)(unsafe.Pointer(addr))

	prot, err := protect(addr, unsafe.Sizeof(*slot), protectReadWrite)
	if err != nil {
		return 0, fmt.Errorf("%s %#x: %w", protectName, addr, err)
	}

	old := *slot
	*slot = value

	_, err = protect(addr, unsafe.Sizeof(*slot), prot)
	if err != nil {
		return old, fmt.Errorf("%s restore %#x: %w", protectName, addr, err)
	}

	return old, nil
}

func FindPattern(module uintptr, image []byte, p pattern.Pattern) []uintptr {
	found := []uintptr{}

	for begin, end := range MappedRanges(image) {
		for offset := begin; offset < end; {
			i := p.Index(image[offset:end])
			if i < 0 {
				break
			}

			found = append(found, module+offset+uintptr(i))
			offset += uintptr(i) + 1
		}
	}

	return found
}

func FindUniquePattern(module uintptr, image []byte, p pattern.Pattern) (uintptr, error) {
	found := FindPattern(module, image, p)

	switch len(found) {
	case 0:
		return 0, ErrPatternNotFound
	case 1:
		return found[0], nil
	default:
		return 0, ErrAmbiguousMatch
	}
}

func FindUniqueFunction(module uintptr, image []byte, anchor pattern.Pattern) (uintptr, error) {
	hit, err := FindUniquePattern(module, image, anchor)
	if err != nil {
		return 0, err
	}

	return FunctionStart(module, image, hit)
}

func ReferencedAddressAt(module uintptr, image []byte, offset uintptr) (address uintptr, length int, ok bool) {
	inst, err := x86asm.Decode(image[offset:], 64)
	if err != nil {
		return 0, 0, false
	}

	if inst.PCRel == 0 || !isRIPMem(inst) {
		return 0, inst.Len, false
	}

	field := image[offset+uintptr(inst.PCRelOff) : offset+uintptr(inst.PCRelOff+inst.PCRel)]

	return module + offset + uintptr(inst.Len) + uintptr(signExtend(field)), inst.Len, true
}

func DecodeRelativeJumpAt(module uintptr, image []byte, offset uintptr) (target uintptr, length int, ok bool) {
	inst, err := x86asm.Decode(image[offset:], 64)
	if err != nil {
		return 0, 0, false
	}

	if inst.Op != x86asm.JMP || inst.PCRel == 0 || isRIPMem(inst) {
		return 0, inst.Len, false
	}

	field := image[offset+uintptr(inst.PCRelOff) : offset+uintptr(inst.PCRelOff+inst.PCRel)]

	return module + offset + uintptr(inst.Len) + uintptr(signExtend(field)), inst.Len, true
}

func FindUniqueReferenceTo(module uintptr, image []byte, address uintptr) (uintptr, error) {
	found := uintptr(0)

	for begin, end := range Functions(image) {
		for offset := begin; offset < end; {
			referenced, length, ok := ReferencedAddressAt(module, image, offset)
			if length == 0 {
				break
			}

			if ok && referenced == address {
				if found != 0 {
					return 0, ErrAmbiguousMatch
				}

				found = module + offset
			}

			offset += uintptr(length)
		}
	}

	if found == 0 {
		return 0, ErrReferenceNotFound
	}

	return found, nil
}
