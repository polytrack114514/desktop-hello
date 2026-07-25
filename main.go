package main

import (
	"syscall"
	"unsafe"
)

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	procMessageBox = user32.NewProc("MessageBoxW")
)

const (
	mbOK       = 0x00000000
	mbIconInfo = 0x00000040
)

func messageBox(hwnd uintptr, caption, title string, flags uint) int {
	captionPtr, _ := syscall.UTF16PtrFromString(caption)
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	ret, _, _ := procMessageBox.Call(
		hwnd,
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(flags),
	)
	return int(ret)
}

func main() {
	messageBox(0,
		"Hello from GitHub!\n\nBuilt on Linux with Go, cross-compiled to Windows.\nDouble-click the .exe to see this dialog.",
		"Desktop Demo", mbOK|mbIconInfo)
}
