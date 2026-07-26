package main

import (
	"runtime"
)

func main() {
	runtime.LockOSThread()

	loadSettings()
	initFonts()
	initIcons()

	hwnd := createMainWindow()
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(hwnd, SW_SHOW)

	messageLoop()
}
