package main

import (
	"syscall"
	"unsafe"
)

// Windows 消息常量（鼠标按钮消息只用于钩子回调判定，不放这里）
const (
	WM_KEYDOWN     = 0x0100
	WM_KEYUP       = 0x0101
	WM_SYSKEYDOWN  = 0x0104
	WM_SYSKEYUP    = 0x0105

	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_RBUTTONDOWN = 0x0204
	WM_RBUTTONUP   = 0x0205
	WM_MBUTTONDOWN = 0x0207
	WM_MBUTTONUP   = 0x0208

	WH_KEYBOARD_LL = 13
	WH_MOUSE_LL    = 14

	WM_USER_KEYEVENT = 0x0400 + 1 // 自定义消息：键盘事件
	WM_USER_MSEVENT  = 0x0400 + 2 // 自定义消息：鼠标事件
)

// KBDLLHOOKSTRUCT 低级键盘钩子结构
type kbdLLHook struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

// MSLLHOOKSTRUCT 低级鼠标钩子结构
type msLLHook struct {
	Pt          point
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type point struct {
	X, Y int32
}

var (
	hInst            = syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW")
	user32           = syscall.NewLazyDLL("user32.dll")
	procSetHook      = user32.NewProc("SetWindowsHookExW")
	procUnhook       = user32.NewProc("UnhookWindowsHookEx")
	procCallNextHook = user32.NewProc("CallNextHookEx")
	procPostMessage  = user32.NewProc("PostMessageW")
)

var (
	mainHwnd    uintptr
	kbHookHnd   uintptr
	mouseHookHnd uintptr
)

// keyEvent 通过 WM_USER_KEYEVENT 投递到主线程窗口的 wParam
// 高位：1=按下 0=松开；低位：VK 码
type keyEvent struct {
	vk   uint16
	down bool
}

// mouseEvent 通过 WM_USER_MSEVENT 投递
// down: 按下；button: 0=左 1=中 2=右
type mouseEvent struct {
	button uint8
	down   bool
}

// installHooks 安装全局键盘和鼠标低级钩子
func installHooks(hwnd uintptr) {
	mainHwnd = hwnd

	kbProc := syscall.NewCallback(keyboardHookProc)
	mouseProc := syscall.NewCallback(mouseHookProc)

	hMod, _, _ := hInst.Call(0)

	r1, _, _ := procSetHook.Call(WH_KEYBOARD_LL, kbProc, hMod, 0)
	kbHookHnd = r1

	r2, _, _ := procSetHook.Call(WH_MOUSE_LL, mouseProc, hMod, 0)
	mouseHookHnd = r2
}

func uninstallHooks() {
	if kbHookHnd != 0 {
		procUnhook.Call(kbHookHnd)
	}
	if mouseHookHnd != 0 {
		procUnhook.Call(mouseHookHnd)
	}
}

// keyboardHookProc 系统注入线程调用，仅读取参数并 PostMessage 到主线程
func keyboardHookProc(code int, wParam uintptr, lParam uintptr) uintptr {
	if code >= 0 && mainHwnd != 0 {
		kb := (*kbdLLHook)(unsafe.Pointer(lParam))
		down := false
		switch wParam {
		case WM_KEYDOWN, WM_SYSKEYDOWN:
			down = true
		case WM_KEYUP, WM_SYSKEYUP:
			down = false
		default:
			return callNext(code, wParam, lParam)
		}
		ev := keyEvent{vk: uint16(kb.VkCode), down: down}
		// 把 ev 编码到 wParam：高 16 位 down 标志，低 16 位 vk
		w := uintptr(uint32(ev.vk) | boolToU32(ev.down)<<16)
		// lParam 复用做去重计数（这里简单用 0）
		procPostMessage.Call(mainHwnd, WM_USER_KEYEVENT, w, 0)
	}
	return callNext(code, wParam, lParam)
}

func mouseHookProc(code int, wParam uintptr, lParam uintptr) uintptr {
	if code >= 0 && mainHwnd != 0 {
		var button uint8
		var down bool
		got := true
		switch wParam {
		case WM_LBUTTONDOWN:
			button, down = 0, true
		case WM_LBUTTONUP:
			button, down = 0, false
		case WM_MBUTTONDOWN:
			button, down = 1, true
		case WM_MBUTTONUP:
			button, down = 1, false
		case WM_RBUTTONDOWN:
			button, down = 2, true
		case WM_RBUTTONUP:
			button, down = 2, false
		default:
			got = false
		}
		if got {
			_ = lParam // 不用鼠标位置
			w := uintptr(uint32(button) | boolToU32(down)<<16)
			procPostMessage.Call(mainHwnd, WM_USER_MSEVENT, w, 0)
		}
	}
	return callNext(code, wParam, lParam)
}

func callNext(code int, wParam, lParam uintptr) uintptr {
	r, _, _ := procCallNextHook.Call(0, uintptr(code), wParam, lParam)
	return r
}

func boolToU32(b bool) uint32 {
	if b {
		return 1
	}
	return 0
}

// decodeKeyEvent 从 wParam 解码 keyEvent
func decodeKeyEvent(w uintptr) keyEvent {
	v := uint32(w)
	return keyEvent{
		vk:   uint16(v & 0xFFFF),
		down: (v>>16)&1 == 1,
	}
}

// decodeMouseEvent 从 wParam 解码 mouseEvent
func decodeMouseEvent(w uintptr) mouseEvent {
	v := uint32(w)
	return mouseEvent{
		button: uint8(v & 0xFFFF),
		down:   (v>>16)&1 == 1,
	}
}
