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

const Version = "0.7.2"

// i18n 多语言文本
func tr(s string) string {
	if settings.Language == 1 {
		return trEn(s)
	}
	return s
}

func trEn(s string) string {
	switch s {
	case "设置":
		return "Settings"
	case "外观":
		return "Appearance"
	case "透明度":
		return "Opacity"
	case "面板大小":
		return "Panel Size"
	case "主题":
		return "Theme"
	case "暗色":
		return "Dark"
	case "亮色":
		return "Light"
	case "语言":
		return "Language"
	case "中文":
		return "Chinese"
	case "英文":
		return "English"
	case "前往官网":
		return "Website"
	case "退出":
		return "Exit"
	case "设置自动保存":
		return "Auto-saved"
	}
	return s
}

// Theme 主题配色
type Theme struct {
	Name   string
	Accent [3]uint8 // 按键按下时的颜色
	KeyOn  [3]uint8 // 按键标签颜色（按下时）
	KeyOff [3]uint8 // 按键标签颜色（未按下）
}

var themes = []Theme{
	{Name: "橙色", Accent: [3]uint8{255, 165, 0}, KeyOn: [3]uint8{255, 255, 255}, KeyOff: [3]uint8{220, 220, 220}},
}

func currentTheme() Theme {
	return themes[0]
}

// ColorScheme 暗色/亮色配色方案
type ColorScheme struct {
	Bg       [3]uint8 // 面板背景
	KeyOff   [3]uint8 // 按键未按下底色
	KeyLabel [3]uint8 // 按键标签色
	KeyLabelOn [3]uint8 // 按键按下时标签色
	MouseBg  [3]uint8 // 鼠标区背景
	CloseBg  [3]uint8 // 关闭按钮底色
}

var darkScheme = ColorScheme{
	Bg:       [3]uint8{0, 0, 0},
	KeyOff:   [3]uint8{60, 60, 60},
	KeyLabel: [3]uint8{220, 220, 220},
	KeyLabelOn: [3]uint8{255, 255, 255},
	MouseBg:  [3]uint8{60, 60, 60},
	CloseBg:  [3]uint8{200, 50, 50},
}

var lightScheme = ColorScheme{
	Bg:       [3]uint8{240, 240, 240},
	KeyOff:   [3]uint8{210, 210, 210},
	KeyLabel: [3]uint8{60, 60, 60},
	KeyLabelOn: [3]uint8{255, 255, 255},
	MouseBg:  [3]uint8{210, 210, 210},
	CloseBg:  [3]uint8{220, 80, 80},
}

func currentScheme() ColorScheme {
	if settings.Mode == 1 {
		return lightScheme
	}
	return darkScheme
}

type Settings struct {
	Alpha    int // 透明度百分比 30-100
	Scale    int // 面板大小百分比 50-100
	Mode     int // 0=暗色, 1=亮色
	Language int // 0=中文, 1=英文
}

var settings = Settings{
	Alpha:    70,
	Scale:    100,
	Mode:     0,
	Language: 0,
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
		case "mode":
			if n, err := strconv.Atoi(val); err == nil {
				settings.Mode = clampInt(n, 0, 1)
			}
		case "language":
			if n, err := strconv.Atoi(val); err == nil {
				settings.Language = clampInt(n, 0, 1)
			}
		}
	}
}

func saveSettings() {
	data := "alpha=" + strconv.Itoa(settings.Alpha) +
		"\nscale=" + strconv.Itoa(settings.Scale) +
		"\nmode=" + strconv.Itoa(settings.Mode) +
		"\nlanguage=" + strconv.Itoa(settings.Language) + "\n"
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
	procGetDlgCtrlID    = user32.NewProc("GetDlgCtrlID")
	procSetBkColor      = gdi32.NewProc("SetBkColor")

	hwndSettings   uintptr
	hwndAlphaEdit  uintptr
	hwndSizeEdit   uintptr
	hwndDarkBtn    uintptr
	hwndLightBtn   uintptr
	hwndCnBtn      uintptr
	hwndEnBtn      uintptr
	controlsCreated bool

	hbrWindowBg    uintptr
	hbrCardBg      uintptr
	hbrEditBg      uintptr
	hbrBtnSelected uintptr
	hbrBtnNormal   uintptr
)

const (
	idAlphaEdit  = 1001
	idSizeEdit   = 1002
	idDarkBtn    = 1012
	idLightBtn   = 1013
	idCnBtn      = 1014
	idEnBtn      = 1015
)

const (
	wsDialogFrame = 0x00400000
	wsCaption     = 0x00C00000
	wsSysMenu     = 0x00080000
	wsChild       = 0x40000000
	wsVisible     = 0x10000000
	esNumber      = 0x2000
	bsAutoCheck   = 0x00000003 // BS_AUTOCHECKBOX
	ssNotify      = 0x0100
	ssCenter      = 0x01
	stnClicked    = 0
	mfSeparator   = 0x00000800
	tpmRightButton = 0x0002
	tpmReturnCmd   = 0x0100
)

const (
	WM_ERASEBKGND     = 0x0014
	WM_CTLCOLORSTATIC = 0x0138
	WM_CTLCOLOREDIT   = 0x0133
)

const WM_USER_TRAY = 0x0400 + 3
const WM_COMMAND_ = 0x0111
const EN_CHANGE = 0x0300
const BN_CLICKED = 0
const BM_GETCHECK = 0x00F0
const BM_SETCHECK = 0x00F1
const BST_CHECKED = 1

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
			makeBrushes()
			createSettingsControls(hwnd)
			controlsCreated = true
		}
		return 0

	case WM_ERASEBKGND:
		hdc := wParam
		rc := rect{0, 0, int32(settingsWinW), int32(settingsWinH)}
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rc)), hbrWindowBg)
		card1 := rect{20, 20, 400, 140}
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&card1)), hbrCardBg)
		card2 := rect{20, 155, 400, 235}
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&card2)), hbrCardBg)
		card3 := rect{20, 240, 400, 320}
		procFillRect.Call(hdc, uintptr(unsafe.Pointer(&card3)), hbrCardBg)
		return 1

	case WM_CTLCOLORSTATIC:
		ctrlID, _, _ := procGetDlgCtrlID.Call(lParam)
		hdc := wParam
		switch ctrlID {
		case idDarkBtn:
			if settings.Mode == 0 {
				procSetBkColor.Call(hdc, rgb(0, 120, 212))
				procSetTextColor.Call(hdc, rgb(255, 255, 255))
				return hbrBtnSelected
			}
			procSetBkColor.Call(hdc, rgb(60, 60, 60))
			procSetTextColor.Call(hdc, rgb(136, 136, 136))
			return hbrBtnNormal
		case idLightBtn:
			if settings.Mode == 1 {
				procSetBkColor.Call(hdc, rgb(0, 120, 212))
				procSetTextColor.Call(hdc, rgb(255, 255, 255))
				return hbrBtnSelected
			}
			procSetBkColor.Call(hdc, rgb(60, 60, 60))
			procSetTextColor.Call(hdc, rgb(136, 136, 136))
			return hbrBtnNormal
		case idCnBtn:
			if settings.Language == 0 {
				procSetBkColor.Call(hdc, rgb(0, 120, 212))
				procSetTextColor.Call(hdc, rgb(255, 255, 255))
				return hbrBtnSelected
			}
			procSetBkColor.Call(hdc, rgb(60, 60, 60))
			procSetTextColor.Call(hdc, rgb(136, 136, 136))
			return hbrBtnNormal
		case idEnBtn:
			if settings.Language == 1 {
				procSetBkColor.Call(hdc, rgb(0, 120, 212))
				procSetTextColor.Call(hdc, rgb(255, 255, 255))
				return hbrBtnSelected
			}
			procSetBkColor.Call(hdc, rgb(60, 60, 60))
			procSetTextColor.Call(hdc, rgb(136, 136, 136))
			return hbrBtnNormal
		default:
			procSetBkMode.Call(hdc, transp)
			procSetTextColor.Call(hdc, rgb(224, 224, 224))
			return 0
		}

	case WM_CTLCOLOREDIT:
		hdc := wParam
		procSetBkColor.Call(hdc, rgb(45, 45, 48))
		procSetTextColor.Call(hdc, rgb(224, 224, 224))
		return hbrEditBg

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
		if notify == stnClicked {
			switch ctrlID {
			case idDarkBtn:
				settings.Mode = 0
				saveSettings()
				procInvalidateRect.Call(hwndMain, 0, 0)
				updateThemeButtons()
			case idLightBtn:
				settings.Mode = 1
				saveSettings()
				procInvalidateRect.Call(hwndMain, 0, 0)
				updateThemeButtons()
			case idCnBtn:
				settings.Language = 0
				saveSettings()
				rebuildSettingsUI()
			case idEnBtn:
				settings.Language = 1
				saveSettings()
				rebuildSettingsUI()
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
	createStatic(hwnd, tr("外观"), 32, 28, 100, 24)
	createStatic(hwnd, tr("透明度"), 44, 64, 60, 20)
	hwndAlphaEdit = createEdit(hwnd, idAlphaEdit, 114, 60, 60, 24)
	setWindowText(hwndAlphaEdit, strconv.Itoa(settings.Alpha))
	createStatic(hwnd, "%", 180, 64, 20, 20)

	createStatic(hwnd, tr("面板大小"), 44, 99, 60, 20)
	hwndSizeEdit = createEdit(hwnd, idSizeEdit, 114, 95, 60, 24)
	setWindowText(hwndSizeEdit, strconv.Itoa(settings.Scale))
	createStatic(hwnd, "%", 180, 99, 20, 20)

	createStatic(hwnd, tr("主题"), 32, 163, 100, 24)
	hwndDarkBtn = createStaticBtn(hwnd, idDarkBtn, 44, 195, 80, 28, tr("暗色"))
	hwndLightBtn = createStaticBtn(hwnd, idLightBtn, 134, 195, 80, 28, tr("亮色"))

	createStatic(hwnd, tr("语言"), 32, 248, 100, 24)
	hwndCnBtn = createStaticBtn(hwnd, idCnBtn, 44, 280, 80, 28, tr("中文"))
	hwndEnBtn = createStaticBtn(hwnd, idEnBtn, 134, 280, 80, 28, tr("英文"))

	createStatic(hwnd, "v"+Version, 328, 310, 80, 20)
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

func createCheckbox(parent uintptr, id uintptr, x, y, w, h int) uintptr {
	className, _ := syscall.UTF16PtrFromString("button")
	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)),
		0,
		uintptr(wsChild|wsVisible|bsAutoCheck),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, id, 0, 0)
	return hwnd
}

func setCheckboxText(hwnd uintptr, text string) {
	title, _ := syscall.UTF16PtrFromString(text)
	procSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(title)))
}

func setWindowText(hwnd uintptr, text string) {
	title, _ := syscall.UTF16PtrFromString(text)
	procSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(title)))
}

func makeBrushes() {
	hbrWindowBg, _, _ = procCreateSolidBrush.Call(rgb(26, 26, 26))
	hbrCardBg, _, _ = procCreateSolidBrush.Call(rgb(37, 37, 37))
	hbrEditBg, _, _ = procCreateSolidBrush.Call(rgb(45, 45, 48))
	hbrBtnSelected, _, _ = procCreateSolidBrush.Call(rgb(0, 120, 212))
	hbrBtnNormal, _, _ = procCreateSolidBrush.Call(rgb(60, 60, 60))
}

func updateThemeButtons() {
	procInvalidateRect.Call(hwndDarkBtn, 0, 1)
	procInvalidateRect.Call(hwndLightBtn, 0, 1)
}

func updateLangButtons() {
	procInvalidateRect.Call(hwndCnBtn, 0, 1)
	procInvalidateRect.Call(hwndEnBtn, 0, 1)
}

func rebuildSettingsUI() {
	updateLangButtons()
	setWindowText(hwndDarkBtn, tr("暗色"))
	setWindowText(hwndLightBtn, tr("亮色"))
	setWindowText(hwndCnBtn, tr("中文"))
	setWindowText(hwndEnBtn, tr("英文"))
	procInvalidateRect.Call(hwndSettings, 0, 1)
}

func createStaticBtn(parent uintptr, id uintptr, x, y, w, h int, text string) uintptr {
	className, _ := syscall.UTF16PtrFromString("static")
	title, _ := syscall.UTF16PtrFromString(text)
	hwnd, _, _ := procCreateWindowEx.Call(0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(wsChild|wsVisible|ssNotify|ssCenter),
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		parent, id, 0, 0)
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

const settingsWinW = 420
const settingsWinH = 420

func createSettingsWindow() uintptr {
	className, _ := syscall.UTF16PtrFromString("BongoKeySettingsClass")
	title, _ := syscall.UTF16PtrFromString(tr("设置"))

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
		0, 0, settingsWinW, settingsWinH,
		0, 0, hMod, 0,
	)
	return r
}

func showSettingsWindow() {
	if hwndSettings == 0 {
		hwndSettings = createSettingsWindow()
	}
	if hwndSettings != 0 {
		setWindowText(hwndAlphaEdit, strconv.Itoa(settings.Alpha))
		setWindowText(hwndSizeEdit, strconv.Itoa(settings.Scale))
		updateThemeButtons()
		updateLangButtons()

		sw, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
		sh, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
		x := (int(sw) - settingsWinW) / 2
		y := (int(sh) - settingsWinH) / 2
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
