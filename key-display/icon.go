package main

var (
	procLoadImage = user32.NewProc("LoadImageW")
)

const (
	IMAGE_ICON   = 1
	LR_DEFAULTSIZE = 0x00000040
	LR_SHARED      = 0x00008000
)

var appIcon uintptr
var appIconSmall uintptr

func initIcons() {
	hMod, _, _ := hInst.Call(0)

	makeIntResource := func(id uintptr) uintptr {
		return id
	}

	appIcon, _, _ = procLoadImage.Call(
		hMod,
		makeIntResource(1),
		uintptr(IMAGE_ICON),
		32, 32,
		LR_SHARED,
	)
	if appIcon == 0 {
		appIcon, _, _ = procLoadIcon.Call(0, idiApplication)
	}

	appIconSmall, _, _ = procLoadImage.Call(
		hMod,
		makeIntResource(1),
		uintptr(IMAGE_ICON),
		16, 16,
		LR_SHARED,
	)
	if appIconSmall == 0 {
		appIconSmall = appIcon
	}
}
