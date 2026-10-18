package hook

import (
	"errors"
	"fmt"
	"slices"

	"golang.org/x/arch/x86/x86asm"
)

var (
	ErrShortFunction = errors.New("function shorter than patch")
	ErrUnrelocatable = errors.New("instruction cannot be relocated")
)

const PatchSize = 5

type Boundary struct {
	Src int
	Dst int
}

type Relocation struct {
	Code       []byte     // relocated instructions
	SrcLen     int        // source bytes consumed (>= PatchSize)
	Boundaries []Boundary // instruction starts, source order
}

func Relocate(src []byte, srcAddr, dstAddr uintptr) (Relocation, error) {
	var (
		out        []byte
		boundaries []Boundary
		srcOff     int
	)

	for srcOff < PatchSize {
		if srcOff >= len(src) {
			return Relocation{}, ErrShortFunction
		}

		inst, err := x86asm.Decode(src[srcOff:], 64)
		if err != nil {
			return Relocation{}, fmt.Errorf("decode at +%#x: %w", srcOff, err)
		}

		raw := src[srcOff : srcOff+inst.Len]
		dstOff := len(out)

		relocated, err := relocateInst(inst, raw, srcAddr, srcOff, dstAddr, dstOff)
		if err != nil {
			return Relocation{}, err
		}

		boundaries = append(boundaries, Boundary{Src: srcOff, Dst: dstOff})
		out = append(out, relocated...)
		srcOff += inst.Len
	}

	return Relocation{Code: out, SrcLen: srcOff, Boundaries: boundaries}, nil
}

func relocateInst(inst x86asm.Inst, raw []byte, srcAddr uintptr, srcOff int, dstAddr uintptr, dstOff int) ([]byte, error) {
	if inst.PCRel == 0 {
		return append([]byte(nil), raw...), nil
	}

	// Absolute target: field is relative to the end of the instruction, which
	// is also the RIP value for a RIP-relative memory operand.
	target := srcAddr + uintptr(srcOff+inst.Len) + uintptr(signExtend(raw[inst.PCRelOff:inst.PCRelOff+inst.PCRel]))

	if isRIPMem(inst) {
		out := append([]byte(nil), raw...)
		disp := int64(target) - (int64(dstAddr) + int64(dstOff) + int64(inst.Len))
		if disp != int64(int32(disp)) {
			return nil, fmt.Errorf("%w: RIP displacement out of range", ErrUnrelocatable)
		}

		putInt32(out[inst.PCRelOff:], int32(disp))

		return out, nil
	}

	return relocateBranch(inst, raw, target, dstAddr, dstOff)
}

func relocateBranch(inst x86asm.Inst, raw []byte, target, dstAddr uintptr, dstOff int) ([]byte, error) {
	switch inst.Op {
	case x86asm.LOOP, x86asm.LOOPE, x86asm.LOOPNE, x86asm.JCXZ, x86asm.JECXZ, x86asm.JRCXZ:
		return nil, fmt.Errorf("%w: %s has no 32-bit form", ErrUnrelocatable, inst.Op)
	}

	var (
		head   []byte // prefixes plus opcode of the output form
		outLen int
	)

	switch inst.PCRel {
	case 4: // already rel32: keep bytes up to the field
		head = raw[:inst.PCRelOff]
		outLen = inst.Len
	case 1: // rel8: widen to rel32
		prefixes := raw[:inst.PCRelOff-1]
		opcode := raw[inst.PCRelOff-1]

		if inst.Op == x86asm.JMP {
			head = append(append([]byte(nil), prefixes...), 0xE9)
		} else {
			head = append(append([]byte(nil), prefixes...), 0x0F, 0x80|(opcode&0x0F))
		}

		outLen = len(head) + 4
	default:
		return nil, fmt.Errorf("%w: %d-byte relative field", ErrUnrelocatable, inst.PCRel)
	}

	disp := int64(target) - (int64(dstAddr) + int64(dstOff) + int64(outLen))
	if disp != int64(int32(disp)) {
		return nil, fmt.Errorf("%w: branch displacement out of range", ErrUnrelocatable)
	}

	out := append(append([]byte(nil), head...), 0, 0, 0, 0)
	putInt32(out[len(out)-4:], int32(disp))

	return out, nil
}

func isRIPMem(inst x86asm.Inst) bool {
	for _, arg := range inst.Args {
		if arg == nil {
			break
		}

		mem, ok := arg.(x86asm.Mem)
		if ok && mem.Base == x86asm.RIP {
			return true
		}
	}

	return false
}

func signExtend(field []byte) int64 {
	var v int64

	for _, b := range slices.Backward(field) {
		v = v<<8 | int64(b)
	}

	shift := uint(64 - 8*len(field))

	return v << shift >> shift
}

func putInt32(dst []byte, v int32) {
	dst[0] = byte(v)
	dst[1] = byte(v >> 8)
	dst[2] = byte(v >> 16)
	dst[3] = byte(v >> 24)
}
