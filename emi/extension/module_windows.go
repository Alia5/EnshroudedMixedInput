package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

const moduleHandleFlags = windows.GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS |
	windows.GET_MODULE_HANDLE_EX_FLAG_UNCHANGED_REFCOUNT

func modulePath() (string, error) {
	var module windows.Handle

	anchor := uintptr(unsafe.Pointer(&moduleAnchor))

	err := windows.GetModuleHandleEx(moduleHandleFlags, (*uint16)(unsafe.Pointer(anchor)), &module)
	if err != nil {
		return "", err
	}

	var path [windows.MAX_LONG_PATH]uint16

	n, err := windows.GetModuleFileName(module, &path[0], uint32(len(path)))
	if err != nil {
		return "", err
	}

	return windows.UTF16ToString(path[:n]), nil
}

var moduleAnchor byte
