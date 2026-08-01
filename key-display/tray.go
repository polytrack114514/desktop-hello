package main

import (
	"unsafe"
)

// 托盘图标相关
var (
	procShellNotify = shell32.NewProc("Shell_NotifyIconW")
)

// Shell_NotifyIconW 常量
const (
	nimAdd    = 0x00000000
	nimDelete = 0x00000002
	nifMessage = 0x00000001
	nifIcon    = 0x00000002
	nifTip     = 0x00000004
	idiApplication = 32512
)

// NOTIFYICONDATAW 结构体（64位 Windows）
type notifyIconData struct {
	CbSize           uint32
	_pad1            uint32
	HWnd             uintptr
	UID              uint32
	UFlags           uint32
	UCallbackMessage uint32
	_pad2            uint32
	HIcon            uintptr
	SzTip            [128]uint16
	DwState          uint32
	DwStateMask      uint32
	SzInfo           [256]uint16
	UTimeout         uint32
	SzInfoTitle      [64]uint16
	DwInfoFlags      uint32
	GuidItem         [16]byte
	HBalloonIcon     uintptr
}

func addTrayIcon(hwnd uintptr) {
	var nid notifyIconData
	nid.CbSize = uint32(unsafe.Sizeof(notifyIconData{}))
	nid.HWnd = hwnd
	nid.UID = 1
	nid.UFlags = nifMessage | nifIcon | nifTip
	nid.UCallbackMessage = WM_USER_TRAY
	nid.HIcon = appIcon

	tip := utf16Slice("按键显示器 v" + Version)
	copy(nid.SzTip[:], tip[:min(len(tip), 128)])

	procShellNotify.Call(nimAdd, uintptr(unsafe.Pointer(&nid)))
}

func removeTrayIcon(hwnd uintptr) {
	var nid notifyIconData
	nid.CbSize = uint32(unsafe.Sizeof(notifyIconData{}))
	nid.HWnd = hwnd
	nid.UID = 1
	procShellNotify.Call(nimDelete, uintptr(unsafe.Pointer(&nid)))
}

func showTrayMenu(hwnd uintptr) {
	menu, _, _ := procCreatePopupMenu.Call()

	settingsText := utf16Slice(tr("设置"))
	websiteText := utf16Slice(tr("GitHub 仓库"))
	exitText := utf16Slice(tr("退出"))

	procAppendMenu.Call(menu, 0, cmdSettings, uintptr(unsafe.Pointer(&settingsText[0])))
	procAppendMenu.Call(menu, 0, cmdWebsite, uintptr(unsafe.Pointer(&websiteText[0])))
	procAppendMenu.Call(menu, mfSeparator, 0, 0)
	procAppendMenu.Call(menu, 0, cmdExit, uintptr(unsafe.Pointer(&exitText[0])))

	var pt point
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))

	procSetForeground.Call(hwnd)

	cmd, _, _ := procTrackPopupMenu.Call(
		menu, tpmRightButton|tpmReturnCmd,
		uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)

	procDestroyMenu.Call(menu)

	switch int(cmd) {
	case cmdSettings:
		showSettingsWindow()
	case cmdWebsite:
		openWebsite()
	case cmdExit:
		procDestroyWindow.Call(hwnd)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
