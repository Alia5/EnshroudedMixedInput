package hook

import "sync/atomic"

type TablePatch struct {
	slot     atomic.Uintptr
	original atomic.Uintptr
}

func (p *TablePatch) Apply(slot, original, callback uintptr) error {
	p.original.Store(original)

	_, err := Swap(slot, callback)
	if err != nil {
		p.original.Store(0)

		return err
	}

	p.slot.Store(slot)

	return nil
}

func (p *TablePatch) Installed() bool {
	return p.slot.Load() != 0
}

func (p *TablePatch) Restore() error {
	slot := p.slot.Swap(0)
	if slot == 0 {
		return nil
	}

	_, err := Swap(slot, p.original.Load())

	return err
}

func (p *TablePatch) CallOriginal(args ...uintptr) uintptr {
	original := p.original.Load()
	if original == 0 {
		return 0
	}

	return callNative(original, args...)
}
