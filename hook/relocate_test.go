package hook_test

import (
	"errors"
	"slices"
	"testing"

	"golang.org/x/arch/x86/x86asm"

	"github.com/Alia5/EnshroudedMixedInput/hook"
)

func TestRelocate(t *testing.T) {
	const (
		fakeSourceAddr     = 0x1000
		fakeTrampolineAddr = 0x2000
	)

	cases := []struct {
		name        string
		src         []byte
		expectedLen int
		expectedErr error
	}{
		{
			name:        "SUCCESS_PlainProlog",
			src:         []byte{0x48, 0x89, 0x5C, 0x24, 0x08}, // mov [rsp+8], rbx
			expectedLen: 5,
		},
		{
			name:        "SUCCESS_RIPRelativeLea",
			src:         []byte{0x48, 0x8D, 0x05, 0x20, 0x00, 0x00, 0x00}, // lea rax, [rip+0x20]
			expectedLen: 7,
		},
		{
			name:        "SUCCESS_CallRel32",
			src:         []byte{0xE8, 0x00, 0x10, 0x00, 0x00}, // call rip+0x1000
			expectedLen: 5,
		},
		{
			name:        "SUCCESS_JmpRel8Widened",
			src:         []byte{0xEB, 0x10, 0x90, 0x90, 0x90}, // jmp .+0x10; nops
			expectedLen: 5 + 3,                                // 2-byte jmp -> 5-byte, then three nops
		},
		{
			name:        "SUCCESS_JccRel8Widened",
			src:         []byte{0x74, 0x10, 0x90, 0x90, 0x90}, // je .+0x10; nops
			expectedLen: 6 + 3,                                // 2-byte je -> 6-byte, then three nops
		},
		{
			name:        "SUCCESS_StraddleBoundary",
			src:         []byte{0x48, 0x8B, 0x44, 0x24, 0x48, 0x89}, // mov r64 (5) + start of next
			expectedLen: 0,
		},
		{
			name:        "FAIL_ShortFunction",
			src:         []byte{0xC3}, // ret
			expectedErr: hook.ErrShortFunction,
		},
		{
			name:        "FAIL_LoopNoWideForm",
			src:         []byte{0xE2, 0xFE, 0x90, 0x90, 0x90}, // loop .-2
			expectedErr: hook.ErrUnrelocatable,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := hook.Relocate(c.src, fakeSourceAddr, fakeTrampolineAddr)

			if c.expectedErr != nil {
				if !errors.Is(err, c.expectedErr) {
					t.Fatalf("err = %v, expected %v", err, c.expectedErr)
				}

				return
			}

			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}

			if c.expectedLen != 0 && len(got.Code) != c.expectedLen {
				t.Errorf("len(Code) = %d, expected %d", len(got.Code), c.expectedLen)
			}

			if got.SrcLen < hook.PatchSize {
				t.Errorf("SrcLen = %d, expected >= %d", got.SrcLen, hook.PatchSize)
			}

			assertTargetsPreserved(t, c.src, got, fakeSourceAddr, fakeTrampolineAddr)
		})
	}
}

func assertTargetsPreserved(t *testing.T, src []byte, r hook.Relocation, srcAddr, dstAddr uintptr) {
	t.Helper()

	for _, b := range r.Boundaries {
		si, err := x86asm.Decode(src[b.Src:], 64)
		if err != nil {
			t.Fatalf("decode src +%#x: %v", b.Src, err)
		}

		di, err := x86asm.Decode(r.Code[b.Dst:], 64)
		if err != nil {
			t.Fatalf("decode dst +%#x: %v", b.Dst, err)
		}

		if si.Op != di.Op {
			t.Errorf("+%#x: op %s -> %s", b.Src, si.Op, di.Op)
		}

		if si.PCRel == 0 {
			if got := r.Code[b.Dst : b.Dst+di.Len]; string(got) != string(src[b.Src:b.Src+si.Len]) {
				t.Errorf("+%#x: verbatim bytes changed", b.Src)
			}

			continue
		}

		srcTarget := srcAddr + uintptr(b.Src+si.Len) + uintptr(rel(src[b.Src:], si))
		dstTarget := dstAddr + uintptr(b.Dst+di.Len) + uintptr(rel(r.Code[b.Dst:], di))

		if srcTarget != dstTarget {
			t.Errorf("+%#x: target %#x -> %#x", b.Src, srcTarget, dstTarget)
		}
	}
}

func rel(raw []byte, inst x86asm.Inst) int64 {
	field := raw[inst.PCRelOff : inst.PCRelOff+inst.PCRel]

	var v int64

	for _, b := range slices.Backward(field) {
		v = v<<8 | int64(b)
	}

	shift := uint(64 - 8*len(field))

	return v << shift >> shift
}
