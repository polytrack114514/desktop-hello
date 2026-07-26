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

const Version = "0.3.2"

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
)

const (
	idAlphaEdit  = 1001
	idSizeEdit   = 1002
	idThemeCombo = 1003
)

const (
	cbsDropdownlist = 0x0003
)

const (
	wsDialogFrame = 0x00400000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsChild       = 0x40000000
	wsVisible     = 0x10000000
	esNumber      = 0x0020
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
		createSettingsControls(hwnd)
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
		if notify == CBN_SELCHANGE && ctrlID == idThemeCombo {
			idx, _, _ := procSendMessage.Call(hwndThemeCombo, 0x0147, 0, 0)
			settings.Theme = int(idx)
			saveSettings()
			procInvalidateRect.Call(hwndMain, 0, 0)
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
	createStatic(hwnd, "透明度% (30-100)", 20, 20, 120, 20)
	hwndAlphaEdit = createEdit(hwnd, strconv.Itoa(settings.Alpha), 150, 20, 60, 24, idAlphaEdit)

	createStatic(hwnd, "面板大小% (50-100)", 20, 60, 120, 20)
	hwndSizeEdit = createEdit(hwnd, strconv.Itoa(settings.Scale), 150, 60, 60, 24, idSizeEdit)

	createStatic(hwnd, "主题颜色", 20, 100, 120, 20)
	hwndThemeCombo = createCombo(hwnd, 150, 100, 80, 24, idThemeCombo)

	createStatic(hwnd, "设置自动保存", 20, 140, 120, 20)
}

func createStatic(parent uintptr, text string, x, y, w, h int) uintptr {
	staticClass := utf16Slice("static")
	txt := utf16Slice(text)
	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(&staticClass[0])),
		uintptr(unsafe.Pointer(&txt[0])),
		uintptr(wsChild|wsVisible),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, 0, 0, 0)
	runtime.KeepAlive(staticClass)
	runtime.KeepAlive(txt)
	return hwnd
}

func createEdit(parent uintptr, text string, x, y, w, h int, id uintptr) uintptr {
	editClass := utf16Slice("edit")
	txt := utf16Slice(text)
	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(&editClass[0])),
		uintptr(unsafe.Pointer(&txt[0])),
		uintptr(wsChild|wsVisible|esNumber),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, id, 0, 0)
	runtime.KeepAlive(editClass)
	runtime.KeepAlive(txt)
	return hwnd
}

func createCombo(parent uintptr, x, y, w, h int, id uintptr) uintptr {
	comboClass := utf16Slice("combobox")
	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(&comboClass[0])),
		0,
		uintptr(wsChild|wsVisible|cbsDropdownlist),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h+100),
		parent, id, 0, 0)
	runtime.KeepAlive(comboClass)
	// 填充选项
	for _, t := range themes {
		item := utf16Slice(t.Name)
		procSendMessage.Call(hwnd, 0x0140, 0, uintptr(unsafe.Pointer(&item[0])))
		runtime.KeepAlive(item)
	}
	// 选中当前主题
	procSendMessage.Call(hwnd, 0x014E, uintptr(settings.Theme), 0)
	return hwnd
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
		0, 0, 280, 200,
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
