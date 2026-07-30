package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const Version = "0.4.4"

// Theme 主题配色
type Theme struct {
	Name   string
	Accent [3]uint8 // 按键按下时的颜色
	KeyOn  [3]uint8 // 按键标签颜色（按下时）
	KeyOff [3]uint8 // 按键标签颜色（未按下）
}

var themes = []Theme{
	{Name: "橙色", Accent: [3]uint8{255, 165, 0}, KeyOn: [3]uint8{255, 255, 255}, KeyOff: [3]uint8{220, 220, 220}},
	{Name: "蓝色", Accent: [3]uint8{0, 120, 215}, KeyOn: [3]uint8{255, 255, 255}, KeyOff: [3]uint8{220, 220, 220}},
	{Name: "绿色", Accent: [3]uint8{0, 200, 83}, KeyOn: [3]uint8{255, 255, 255}, KeyOff: [3]uint8{220, 220, 220}},
	{Name: "紫色", Accent: [3]uint8{156, 39, 176}, KeyOn: [3]uint8{255, 255, 255}, KeyOff: [3]uint8{220, 220, 220}},
	{Name: "红色", Accent: [3]uint8{244, 67, 54}, KeyOn: [3]uint8{255, 255, 255}, KeyOff: [3]uint8{220, 220, 220}},
}

func currentTheme() Theme {
	if settings.Theme >= 0 && settings.Theme < len(themes) {
		return themes[settings.Theme]
	}
	return themes[0]
}

type Settings struct {
	Alpha   int // 透明度百分比 30-100
	Scale   int // 面板大小百分比 50-100
	Theme   int // 主题索引
}

var settings = Settings{
	Alpha: 70,
	Scale: 100,
	Theme: 0,
}

// alphaValue 把百分比(30-100)转成 Windows 透明度值(0-255)
func alphaValue() uintptr {
	return uintptr(settings.Alpha * 255 / 100)
}

func scaleFloat() float64 {
	return float64(settings.Scale) / 100.0
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func settingsFilePath() string {
	exe, err := os.Executable()
	if err != nil {
		return "settings.txt"
	}
	return filepath.Join(filepath.Dir(exe), "settings.txt")
}

func loadSettings() {
	data, err := os.ReadFile(settingsFilePath())
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(strings.TrimSpace(line), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		switch key {
		case "alpha":
			if n, err := strconv.Atoi(val); err == nil {
				settings.Alpha = clampInt(n, 30, 100)
			}
		case "scale":
			if n, err := strconv.Atoi(val); err == nil {
				settings.Scale = clampInt(n, 50, 100)
			}
		case "theme":
			if n, err := strconv.Atoi(val); err == nil {
				settings.Theme = clampInt(n, 0, len(themes)-1)
			}
		}
	}
}

func saveSettings() {
	data := "alpha=" + strconv.Itoa(settings.Alpha) +
		"\nscale=" + strconv.Itoa(settings.Scale) +
		"\ntheme=" + strconv.Itoa(settings.Theme) + "\n"
	os.WriteFile(settingsFilePath(), []byte(data), 0644)
}

var (
	shell32             = syscall.NewLazyDLL("shell32.dll")
	procShellExecute    = shell32.NewProc("ShellExecuteW")
	procLoadIcon        = user32.NewProc("LoadIconW")
	procCreatePopupMenu  = user32.NewProc("CreatePopupMenu")
	procAppendMenu       = user32.NewProc("AppendMenuW")
	procTrackPopupMenu  = user32.NewProc("TrackPopupMenu")
	procSetForeground   = user32.NewProc("SetForegroundWindow")
	procGetCursorPos    = user32.NewProc("GetCursorPos")
	procDestroyMenu     = user32.NewProc("DestroyMenu")
	procSetWindowText   = user32.NewProc("SetWindowTextW")
	procSetWindowPos    = user32.NewProc("SetWindowPos")
	procGetDlgItemText  = user32.NewProc("GetDlgItemTextW")

	hwndSettings   uintptr
	hwndAlphaEdit  uintptr
	hwndSizeEdit   uintptr
	hwndThemeCombo uintptr
	controlsCreated bool
)

const (
	idAlphaEdit  = 1001
	idSizeEdit   = 1002
	idThemeCombo = 1003
)

const (
	cbsDropdown = 0x0002
	cbsHasStrings = 0x0040
)

const (
	wsDialogFrame = 0x00400000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsChild       = 0x40000000
	wsVisible     = 0x10000000
	esNumber      = 0x2000
	esLeft        = 0x0000
	mfSeparator   = 0x00000800
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
)

const WM_USER_TRAY = 0x0400 + 3
const WM_COMMAND_ = 0x0111
const EN_CHANGE = 0x0300
const CBN_SELCHANGE = 1

const (
	cmdSettings = 2001
	cmdWebsite  = 2002
	cmdExit     = 2003
)

func utf16Slice(s string) []uint16 {
	buf := make([]uint16, len(s)+1)
	n := 0
	for _, r := range s {
		if r < 0x10000 {
			buf[n] = uint16(r)
			n++
		} else {
			buf[n] = uint16(0xD800 + (r-0x10000)/0x400)
			n++
			buf[n] = uint16(0xDC00 + (r-0x10000)%0x400)
			n++
		}
	}
	buf[n] = 0
	return buf[:n+1]
}

func settingsWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_CREATE:
		if !controlsCreated {
			createSettingsControls(hwnd)
			controlsCreated = true
		}
		return 0

	case WM_COMMAND_:
		ctrlID := uint16(wParam & 0xFFFF)
		notify := uint16((wParam >> 16) & 0xFFFF)
		if notify == EN_CHANGE {
			switch ctrlID {
			case idAlphaEdit:
				text := getEditText(hwnd, idAlphaEdit)
				if n, err := strconv.Atoi(text); err == nil {
					n = clampInt(n, 30, 100)
					if n != settings.Alpha {
						settings.Alpha = n
						procSetLayeredAttr.Call(hwndMain, 0, alphaValue(), LWA_ALPHA)
						saveSettings()
					}
				}
			case idSizeEdit:
				text := getEditText(hwnd, idSizeEdit)
				if n, err := strconv.Atoi(text); err == nil {
					n = clampInt(n, 50, 100)
					if n != settings.Scale {
						settings.Scale = n
						applyScale()
						resizeMainWindow()
						saveSettings()
					}
				}
			}
		}
		if notify == CBN_SELCHANGE {
			if lParam == hwndThemeCombo {
				idx, _, _ := procSendMessage.Call(hwndThemeCombo, 0x0147, 0, 0)
				if idx >= 0 && int(idx) < len(themes) {
					settings.Theme = int(idx)
					saveSettings()
					procInvalidateRect.Call(hwndMain, 0, 0)
				}
			}
		}
		return 0

	case WM_CLOSE:
		procShowWindow.Call(hwnd, 0)
		return 0
	}
	r, _, _ := procDefWindowProc.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

func getEditText(hwnd, ctrlID uintptr) string {
	var buf [32]uint16
	procGetDlgItemText.Call(hwnd, ctrlID, uintptr(unsafe.Pointer(&buf[0])), 32)
	return syscall.UTF16ToString(buf[:])
}

func createSettingsControls(hwnd uintptr) {
	createStatic(hwnd, "透明度", 20, 20, 60, 20)
	hwndAlphaEdit = createEdit(hwnd, idAlphaEdit, 85, 18, 80, 24)
	setWindowText(hwndAlphaEdit, strconv.Itoa(settings.Alpha))

	createStatic(hwnd, "面板大小", 20, 55, 60, 20)
	hwndSizeEdit = createEdit(hwnd, idSizeEdit, 85, 53, 80, 24)
	setWindowText(hwndSizeEdit, strconv.Itoa(settings.Scale))

	createStatic(hwnd, "主题颜色", 20, 90, 60, 20)
	hwndThemeCombo = createCombo(hwnd, idThemeCombo, 85, 88, 120, 24)

	createStatic(hwnd, "当前主题: "+themes[settings.Theme].Name, 20, 120, 150, 20)
	createStatic(hwnd, "设置自动保存", 20, 145, 120, 20)
	createStatic(hwnd, "版本 v"+Version, 180, 145, 80, 20)
}

func createStatic(parent uintptr, text string, x, y, w, h int) uintptr {
	className, _ := syscall.UTF16PtrFromString("static")
	title, _ := syscall.UTF16PtrFromString(text)
	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsChild|wsVisible),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, 0, 0, 0)
	return hwnd
}

func createEdit(parent uintptr, id uintptr, x, y, w, h int) uintptr {
	className, _ := syscall.UTF16PtrFromString("edit")
	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)),
		0,
		uintptr(wsChild|wsVisible|esNumber),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, id, 0, 0)
	return hwnd
}

func createCombo(parent uintptr, id uintptr, x, y, w, h int) uintptr {
	className, _ := syscall.UTF16PtrFromString("combobox")
	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)),
		0,
		uintptr(wsChild|wsVisible|cbsDropdown|cbsHasStrings),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h+150),
		parent, id, 0, 0)
	// 先清空
	procSendMessage.Call(hwnd, 0x014B, 0, 0) // CB_RESETCONTENT
	// 添加选项 - 用变量保存指针防止 GC 回收
	items := make([][]uint16, len(themes))
	for i, t := range themes {
		items[i] = utf16Slice(t.Name)
		procSendMessage.Call(hwnd, 0x0140, 0, uintptr(unsafe.Pointer(&items[i][0])))
	}
	runtime.KeepAlive(items)
	// 选中当前主题
	procSendMessage.Call(hwnd, 0x014E, uintptr(settings.Theme), 0)
	return hwnd
}

func setWindowText(hwnd uintptr, text string) {
	title, _ := syscall.UTF16PtrFromString(text)
	procSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(title)))
}

func resizeMainWindow() {
	sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	windowX = (int(sw) - panelW) / 2
	windowY = int(sh) - panelH - 40
	procSetWindowPos.Call(hwndMain, 0,
		uintptr(windowX), uintptr(windowY),
		uintptr(panelW), uintptr(panelH),
		0x0004)
	procInvalidateRect.Call(hwndMain, 0, 0)
}

func createSettingsWindow() uintptr {
	className, _ := syscall.UTF16PtrFromString("BongoKeySettingsClass")
	title, _ := syscall.UTF16PtrFromString("设置")

	hMod, _, _ := hInst.Call(0)

	wc := wndClassEx{
		Size:      uint32(unsafe.Sizeof(wndClassEx{})),
		Style:     CS_HREDRAW | CS_VREDRAW,
		WndProc:   syscall.NewCallback(settingsWndProc),
		Instance:  hMod,
		Background: 0,
		ClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	style := uintptr(wsDialogFrame | wsCaption | wsSysMenu)
	r, _, _ := procCreateWindowEx.Call(
		WS_EX_TOPMOST,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		style,
		0, 0, 280, 220,
		0, 0, hMod, 0,
	)
	return r
}

func showSettingsWindow() {
	if hwndSettings == 0 {
		hwndSettings = createSettingsWindow()
	}
	if hwndSettings != 0 {
		sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
		sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
		x := (int(sw) - 280) / 2
		y := (int(sh) - 200) / 2
		procSetWindowPos.Call(hwndSettings, 0,
			uintptr(x), uintptr(y),
			0, 0, 0x0001)
		procShowWindow.Call(hwndSettings, SW_SHOW)
	}
}

func openWebsite() {
	openStr := utf16Slice("open")
	urlStr := utf16Slice("https://github.com/polytrack114514/desktop-hello-")
	procShellExecute.Call(0,
		uintptr(unsafe.Pointer(&openStr[0])),
		uintptr(unsafe.Pointer(&urlStr[0])),
		0, 0, 1)
	runtime.KeepAlive(openStr)
	runtime.KeepAlive(urlStr)
}
