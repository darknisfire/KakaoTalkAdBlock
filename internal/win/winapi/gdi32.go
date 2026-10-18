package winapi

import (
	"syscall"

	"golang.org/x/sys/windows"
)

var gdi32 = windows.NewLazySystemDLL("gdi32.dll")
var createRoundRectRgn = gdi32.NewProc("CreateRoundRectRgn")
var deleteObject = gdi32.NewProc("DeleteObject")

func CreateRoundRectRgn(left, top, right, bottom, ellipseWidth, ellipseHeight int32) uintptr {
	r, _, _ := syscall.SyscallN(createRoundRectRgn.Addr(), uintptr(left), uintptr(top), uintptr(right), uintptr(bottom), uintptr(ellipseWidth), uintptr(ellipseHeight))
	return r
}

func DeleteObject(object uintptr) bool {
	r, _, _ := syscall.SyscallN(deleteObject.Addr(), object)
	return r != 0
}
