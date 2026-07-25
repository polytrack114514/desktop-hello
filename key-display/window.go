package main

import (
	"syscall"
	"time"
	"unsafe"
)

// Windows 窗口相关常量
const (
	WS_POPUP        = 0x80000000
	WS_VISIBLE      = 0x10000000
	WS_CLIPCHILDREN = 0x02000000

	WS_EX_LAYERED     = 0x00080000
	WS_EX_TOPMOST     = 0x00000008
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_TOOLWINDOW  = 0x00000080

	WM_CREATE = 0x0001
	WM_PAINT  = 0x000F
	WM_DESTROY      = 0x0002
	WM_CLOSE        = 0x0010
	WM_LBUTTONDOWN_ = 0x0201 // 不复用 keyhook 里的，避免重名；此处仅文档参考
	WM_NCHITTEST    = 0x0084
	WM_TIMER        = 0x0113
	WM_USER_KEYEVENT_LOCAL = WM_USER_KEYEVENT
	WM_USER_MSEVENT_LOCAL  = WM_USER_MSEVENT

	HTCLIENT    = 1
	HTCAPTION  = 2

	CW_USEDEFAULT = 0x80000000

	LWA_ALPHA = 0x00000002

	IDT_ANIM = 1

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	CS_HREDRAW = 0x0002
	CS_VREDRAW = 0x0001

	SW_SHOW = 5
)

// wndClassEx 简化窗口类结构
type wndClassEx struct {
	Size            uint32
	Style           uint32
	WndProc         uintptr
	ClsExtra        int32
	WndExtra        int32
	Instance        uintptr
	Icon            uintptr
	Cursor          uintptr
	Background       uintptr
	MenuName        *uint16
	ClassName       *uint16
	IconSm          uintptr
}

type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

// 在本文件内别名以避免与 syscall.msg 冲突
type winMsg = msg

var (
	procRegisterClassEx  = user32.NewProc("RegisterClassExW")
	procCreateWindowEx   = user32.NewProc("CreateWindowExW")
	procDefWindowProc    = user32.NewProc("DefWindowProcW")
	procGetMessage       = user32.NewProc("GetMessageW")
	procTranslateMessage = user32.NewProc("TranslateMessage")
	procDispatchMessage  = user32.NewProc("DispatchMessageW")
	procSetLayeredAttr   = user32.NewProc("SetLayeredWindowAttributes")
	procPostQuitMessage  = user32.NewProc("PostQuitMessage")
	procSetTimer         = user32.NewProc("SetTimer")
	procKillTimer        = user32.NewProc("KillTimer")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	procShowWindow       = user32.NewProc("ShowWindow")
)

var hwndMain uintptr

// KeyState 进程内唯一按键状态
type KeyState struct {
	keys      map[uint16]bool // 当前按下的 VK 码
	animStart map[uint16]int64 // 松开回弹开始时间（纳秒），存在即动画中
	left, middle, right bool
}

var state = &KeyState{
	keys:      make(map[uint16]bool),
	animStart: make(map[uint16]int64),
}

// wndProc 窗口过程
func wndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		hwndMain = hwnd
		// 半透明 70%
		procSetLayeredAttr.Call(hwnd, 0, 180, LWA_ALPHA)
		// 安装钩子
		installHooks(hwnd)
		return 0

	case WM_USER_KEYEVENT:
		ev := decodeKeyEvent(wParam)
		applyKey(ev)
		procInvalidateRect.Call(hwnd, 0, 0)
		// 有动画时确保定时器在跑
		if len(state.animStart) > 0 {
			procSetTimer.Call(hwnd, IDT_ANIM, 16, 0)
		}
		return 0

	case WM_USER_MSEVENT:
		ev := decodeMouseEvent(wParam)
		applyMouse(ev)
		procInvalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_TIMER:
		if wParam == IDT_ANIM {
			procInvalidateRect.Call(hwnd, 0, 0)
			if len(state.animStart) == 0 {
				procKillTimer.Call(hwnd, IDT_ANIM)
			}
		}
		return 0

	case WM_PAINT:
		var ps paintStruct
		hdc, _, _ := procBeginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		drawPanel(hdc, state, time.Now().UnixNano())
		procEndPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_NCHITTEST:
		// 鼠标位置
		x := int16(lParam & 0xFFFF)
		y := int16(lParam >> 16)
		// 转换为窗口坐标
		wx := int(x) - windowX
		wy := int(y) - windowY
		// × 按钮区域：可点
		if wx >= panelW-38 && wx <= panelW-8 && wy >= 6 && wy <= 36 {
			return HTCLIENT
		}
		// 其它区域：作为标题栏拖动
		return HTCAPTION

	case WM_LBUTTONDOWN_:
		// 仅 ×按钮区域会进到这（其它被 HTCAPTION 接管）
		procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
		// 触发关闭
		procDestroyWindow.Call(hwnd)
		return 0

	case WM_DESTROY:
		uninstallHooks()
		procPostQuitMessage.Call(0)
		return 0
	}

	r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

// applyKey 应用键盘事件到状态
func applyKey(ev keyEvent) {
	if ev.down {
		state.keys[ev.vk] = true
		delete(state.animStart, ev.vk) // 按下时取消任何松开动画
	} else {
		delete(state.keys, ev.vk)
		state.animStart[ev.vk] = time.Now().UnixNano()
	}
}

// applyMouse 应用鼠标事件到状态
func applyMouse(ev mouseEvent) {
	switch ev.button {
	case 0:
		state.left = ev.down
	case 1:
		state.middle = ev.down
	case 2:
		state.right = ev.down
	}
}

var (
	windowX int
	windowY int
)

// createMainWindow 创建主窗口，返回 hwnd
func createMainWindow() uintptr {
	className, _ := syscall.UTF16PtrFromString("BongoKeyDisplayClass")
	title, _ := syscall.UTF16PtrFromString("Bongo Key Display")

	hMod, _, _ := hInst.Call(0)

	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		Style:     CS_HREDRAW | CS_VREDRAW,
		WndProc:   syscall.NewCallback(wndProc),
		Instance:  hMod,
		Background: 0, // 自己画
		ClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	// 初始位置：屏幕底部居中
	sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	windowX = int(sw) - panelW
	windowX = windowX / 2
	windowY = int(sh) - panelH - 40

	exStyle := uintptr(WS_EX_LAYERED | WS_EX_TOPMOST | WS_EX_TRANSPARENT | WS_EX_TOOLWINDOW)
	style := uintptr(WS_POPUP | WS_VISIBLE | WS_CLIPCHILDREN)

	r1, _, _ := procCreateWindowEx.Call(
		exStyle,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		style,
		uintptr(windowX), uintptr(windowY),
		uintptr(panelW), uintptr(panelH),
		0, 0, hMod, 0,
	)
	return r1
}

// messageLoop 主消息循环
func messageLoop() {
	var m msg
	for {
		r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 {
			break
		}
		procTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

var procDestroyWindow = user32.NewProc("DestroyWindow")
