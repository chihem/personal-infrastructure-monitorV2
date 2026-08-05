//go:build windows

package config

import (
	"syscall"
	"unsafe"
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

var (
	kernel32       = syscall.NewLazyDLL("kernel32.dll")
	moveFileExProc = kernel32.NewProc("MoveFileExW")
)

func replaceFile(oldPath, newPath string) error {
	oldPathPointer, err := syscall.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}
	newPathPointer, err := syscall.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}

	result, _, callErr := moveFileExProc.Call(
		uintptr(unsafe.Pointer(oldPathPointer)),
		uintptr(unsafe.Pointer(newPathPointer)),
		moveFileReplaceExisting|moveFileWriteThrough,
	)
	if result == 0 {
		return callErr
	}
	return nil
}
