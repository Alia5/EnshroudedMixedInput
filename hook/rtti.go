package hook

import (
	"fmt"
	"slices"
	"unsafe"
)

func FindSlot(module uintptr, image []byte, table uintptr, match func([]byte) bool, window uintptr) (uintptr, int, error) {
	for i := 0; ; i++ {
		slot := table + uintptr(i)*unsafe.Sizeof(uintptr(0))

		if !slotReadable(module, image, slot) {
			return 0, 0, fmt.Errorf("no match in %d slots", i)
		}

		method := *(*uintptr)(unsafe.Pointer(slot))

		if !inImage(module, image, method) {
			return 0, 0, fmt.Errorf("no match in %d slots", i)
		}

		target, _, ok := FollowThunk(module, image, method)
		if ok {
			method = target
		}

		if !inImage(module, image, method) {
			continue
		}

		end := min(method-module+window, uintptr(len(image)))

		if match(image[method-module : end]) {
			return slot, i, nil
		}
	}
}

func FindUniqueSlot(
	module uintptr,
	image []byte,
	table uintptr,
	match func([]byte) int,
) (slot, method uintptr, index int, err error) {
	entries := MethodTableEntries(module, image, table)
	if len(entries) == 0 {
		return 0, 0, 0, fmt.Errorf("%w: empty method table", ErrSlotNotFound)
	}

	bodies := methodBodies(module, image, entries)

	start := slices.Min(bodies) - module
	end := sectionEnd(image, slices.Max(bodies)-module)

	region := image[start:end]

	at := match(region)
	if at < 0 {
		return 0, 0, 0, fmt.Errorf("%w: no match in %d slots", ErrSlotNotFound, len(entries))
	}

	hit := module + start + uintptr(at)

	again := match(region[at+1:])
	if again >= 0 {
		return 0, 0, 0, fmt.Errorf("%w: %#x and %#x", ErrAmbiguousMatch, hit, hit+uintptr(again)+1)
	}

	index = -1
	owner := uintptr(0)

	for i, body := range bodies {
		if body <= hit && body >= owner {
			owner = body
			index = i
		}
	}

	if index < 0 {
		return 0, 0, 0, fmt.Errorf("%w: %#x precedes every method", ErrSlotNotFound, hit)
	}

	return table + uintptr(index)*unsafe.Sizeof(uintptr(0)), entries[index], index, nil
}

func methodBodies(module uintptr, image []byte, entries []uintptr) []uintptr {
	bodies := make([]uintptr, len(entries))

	for i, entry := range entries {
		bodies[i] = entry

		target, _, ok := FollowThunk(module, image, entry)
		if ok && inImage(module, image, target) {
			bodies[i] = target
		}
	}

	return bodies
}

func MethodTableEntries(module uintptr, image []byte, table uintptr) []uintptr {
	entries := []uintptr{}

	for i := 0; ; i++ {
		slot := table + uintptr(i)*unsafe.Sizeof(uintptr(0))

		if !slotReadable(module, image, slot) {
			return entries
		}

		entry := *(*uintptr)(unsafe.Pointer(slot))

		if !inImage(module, image, entry) {
			return entries
		}

		entries = append(entries, entry)
	}
}

func slotReadable(module uintptr, image []byte, p uintptr) bool {
	return p >= module && p+unsafe.Sizeof(uintptr(0)) <= module+uintptr(len(image))
}

func SlotOf(module uintptr, image []byte, table, target uintptr) (uintptr, int, error) {
	for i := 0; ; i++ {
		slot := table + uintptr(i)*unsafe.Sizeof(uintptr(0))

		if !slotReadable(module, image, slot) {
			return 0, 0, fmt.Errorf("%w: not found in %d slots", ErrSlotNotFound, i)
		}

		method := *(*uintptr)(unsafe.Pointer(slot))

		if !inImage(module, image, method) {
			return 0, 0, fmt.Errorf("%w: not found in %d slots", ErrSlotNotFound, i)
		}

		if method == target {
			return slot, i, nil
		}
	}
}

func inImage(module uintptr, image []byte, p uintptr) bool {
	return p >= module && p < module+uintptr(len(image))
}
