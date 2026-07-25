package main

import (
	"syscall"
	"unsafe"
)

// GDI / User32 引用
var (
	gdi32                = syscall.NewLazyDLL("gdi32.dll")
	procCreateCompDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procDeleteDC         = gdi32.NewProc("DeleteDC")
	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procSetTextAlign     = gdi32.NewProc("SetTextAlign")
	procTextOut          = gdi32.NewProc("TextOutW")
	procRoundRect        = gdi32.NewProc("RoundRect")
	procEllipse          = gdi32.NewProc("Ellipse")
	procRectangle        = gdi32.NewProc("Rectangle")
	procFillRect         = user32.NewProc("FillRect")
	procBitBlt           = gdi32.NewProc("BitBlt")
	procCreateFontW      = gdi32.NewProc("CreateFontW")
	procBeginPaint       = user32.NewProc("BeginPaint")
	procEndPaint         = user32.NewProc("EndPaint")
	procInvalidateRect   = user32.NewProc("InvalidateRect")
)

const (
	transp      = 1 // TRANSPARENT for SetBkMode
	taCenter    = 6 // TA_CENTER
	taBaseline  = 24
	fwBold      = 700
	defaultPitch = 0
	ffDonotcare = 0x04
	srccopy     = 0x00CC0020
)

// paintStruct Windows PAINTSTRUCT（简化，只需前几字段）
type paintStruct struct {
	Hdc         uintptr
	fErase      int32
	rcPaint     rect
	fRestore    int32
	fIncUpdate  int32
	rgbReserved [32]byte
}

type rect struct {
	Left, Top, Right, Bottom int32
}

// rgb 打包 RGB 成 COLORREF
func rgb(r, g, b uint8) uintptr {
	return uintptr(uint32(r) | uint32(g)<<8 | uint32(b)<<16)
}

var (
	fontKey uintptr // 键名标签字体
	fontX   uintptr // × 按钮字体
)

func initFonts() {
	fontKey = createFontW(14, "Segoe UI", fwBold)
	fontX = createFontW(16, "Segoe UI", fwBold)
}

func createFontW(size int, face string, weight int) uintptr {
	faceUTF16, _ := syscall.UTF16PtrFromString(face)
	r, _, _ := procCreateFontW.Call(
		uintptr(size),               // height
		0, 0, 0,                     // width, escapement, orientation
		uintptr(weight),             // weight
		0, 0, 0, 0, 0, 0, 0, 0,      // italic, underline, ...
		defaultPitch|ffDonotcare,
		uintptr(unsafe.Pointer(faceUTF16)),
	)
	return r
}

// drawPanel 在 hdc 上把整个面板画到 (0,0)~(panelW,panelH)
// state 当前按键状态，animStart 每键动画开始时间戳（纳秒）
func drawPanel(hdc uintptr, st *KeyState, nowNs int64) {
	// 双缓冲：内存 DC
	memDC, _, _ := procCreateCompDC.Call(hdc)
	memBmp, _, _ := procCreateCompBitmap.Call(hdc, uintptr(panelW), uintptr(panelH))
	oldBmp, _, _ := procSelectObject.Call(memDC, memBmp)
	defer func() {
		procSelectObject.Call(memDC, oldBmp)
		procDeleteObject.Call(memBmp)
		procDeleteDC.Call(memDC)
	}()

	// 1) 整面板填充半透明黑色（窗体本身已半透明，这里用纯黑底）
	hbrBg, _, _ := procCreateSolidBrush.Call(rgb(0, 0, 0))
	procFillRect.Call(memDC, uintptr(unsafe.Pointer(&rect{0, 0, panelW, panelH})), hbrBg)
	procDeleteObject.Call(hbrBg)

	// 2) 画每个键
	for _, k := range keyMap {
		drawKey(memDC, k, st, nowNs)
	}

	// 3) 画鼠标示意
	drawMouse(memDC, st, nowNs)

	// 4) 画 × 按钮
	drawCloseButton(memDC)

	// 拷贝到屏幕
	procBitBlt.Call(hdc, 0, 0, uintptr(panelW), uintptr(panelH),
		memDC, 0, 0, srccopy)
}

// drawKey 画单个键，含缩放与颜色插值
func drawKey(hdc uintptr, k KeyDef, st *KeyState, nowNs int64) {
	down := st.keys[k.VK]
	scale, colorR, colorG, colorB := computeKeyVisual(k.VK, down, st, nowNs)

	w := int(float64(k.W) * scale)
	h := int(float64(k.H) * scale)
	x := k.X + (k.W-w)/2
	y := k.Y + (k.H-h)/2

	// 画刷
	hbr, _, _ := procCreateSolidBrush.Call(rgb(colorR, colorG, colorB))
	oldBr, _, _ := procSelectObject.Call(hdc, hbr)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y),
		uintptr(x+w), uintptr(y+h), 8, 8)
	procSelectObject.Call(hdc, oldBr)
	procDeleteObject.Call(hbr)

	// 标签
	procSetBkMode.Call(hdc, transp)
	if down {
		procSetTextColor.Call(hdc, rgb(255, 255, 255))
	} else {
		procSetTextColor.Call(hdc, rgb(220, 220, 220))
	}
	procSelectObject.Call(hdc, fontKey)
	procSetTextAlign.Call(hdc, taCenter|taBaseline)
	drawTextCenter(hdc, k.Label, x+w/2, y+h/2)
}

// computeKeyVisual 计算单键当前缩放与颜色
// 按下：scale=1.1, color=(255,165,0)
// 松开 150ms 内：scale 从 1.1 回到 1.0，颜色从橙回灰
func computeKeyVisual(vk uint16, down bool, st *KeyState, nowNs int64) (scale float64, r, g, b uint8) {
	const animMs = 150
	if down {
		// 按下立即橙 + 放大
		return 1.1, 255, 165, 0
	}
	// 检查是否在松开回弹期
	tStart, ok := st.animStart[vk]
	if !ok {
		// 没动画记录：默认静止
		return 1.0, 60, 60, 60
	}
	elapsedMs := (nowNs - tStart) / int64(1e6)
	if elapsedMs >= animMs {
		delete(st.animStart, vk)
		return 1.0, 60, 60, 60
	}
	t := float64(elapsedMs) / float64(animMs) // 0..1
	scale = 1.1 - 0.1*t
	r = uint8(255 - (255-60)*t)
	g = uint8(165 - (165-60)*t)
	b = uint8(0 + (60-0)*t)
	return
}

// drawMouse 画鼠标示意图
func drawMouse(hdc uintptr, st *KeyState, nowNs int64) {
	// 外轮廓：椭圆体
	// 简化画法：上半部分两个椭圆分区（左/右键），下半部分一个椭圆（中键）
	// 用 procEllipse + procRectangle 组合

	mx := mouseX
	my := mouseY

	// 背景灰底
	hbrBg, _, _ := procCreateSolidBrush.Call(rgb(60, 60, 60))
	oldBr, _, _ := procSelectObject.Call(hdc, hbrBg)
	procRoundRect.Call(hdc, uintptr(mx), uintptr(my),
		uintptr(mx+mouseW), uintptr(my+mouseH), 12, 12)
	procSelectObject.Call(hdc, oldBr)
	procDeleteObject.Call(hbrBg)

	// 左键
	leftColor := colorButton(st.left, st, "left", nowNs)
	rightColor := colorButton(st.right, st, "right", nowNs)
	midColor := colorButton(st.middle, st, "mid", nowNs)

	// 左键区域
	fillRect2(hdc, mx+8, my+8, mx+mouseW/2-2, my+mouseH/2, leftColor)
	// 右键区域
	fillRect2(hdc, mx+mouseW/2+2, my+8, mx+mouseW-8, my+mouseH/2, rightColor)
	// 中键
	hbrMid, _, _ := procCreateSolidBrush.Call(rgbByte(midColor))
	oldMid, _, _ := procSelectObject.Call(hdc, hbrMid)
	procEllipse.Call(hdc,
		uintptr(mx+mouseW/2-14), uintptr(my+mouseH/2+8),
		uintptr(mx+mouseW/2+14), uintptr(my+mouseH/2+40))
	procSelectObject.Call(hdc, oldMid)
	procDeleteObject.Call(hbrMid)
}

func colorButton(down bool, st *KeyState, name string, nowNs int64) [3]uint8 {
	if down {
		return [3]uint8{255, 165, 0}
	}
	// 不做回弹动画，直接返回灰色
	return [3]uint8{60, 60, 60}
}

func fillRect2(hdc uintptr, x1, y1, x2, y2 int, c [3]uint8) {
	hbr, _, _ := procCreateSolidBrush.Call(rgb(c[0], c[1], c[2]))
	procFillRect.Call(hdc, uintptr(unsafe.Pointer(&rect{int32(x1), int32(y1), int32(x2), int32(y2)})), hbr)
	procDeleteObject.Call(hbr)
}

func rgbByte(c [3]uint8) uintptr {
	return rgb(c[0], c[1], c[2])
}

// drawCloseButton 画 × 按钮
func drawCloseButton(hdc uintptr) {
	x := panelW - 38
	y := 6
	w := 30
	h := 30
	hbr, _, _ := procCreateSolidBrush.Call(rgb(200, 50, 50))
	oldBr, _, _ := procSelectObject.Call(hdc, hbr)
	procRoundRect.Call(hdc, uintptr(x), uintptr(y), uintptr(x+w), uintptr(y+h), 6, 6)
	procSelectObject.Call(hdc, oldBr)
	procDeleteObject.Call(hbr)

	procSetBkMode.Call(hdc, transp)
	procSetTextColor.Call(hdc, rgb(255, 255, 255))
	procSelectObject.Call(hdc, fontX)
	procSetTextAlign.Call(hdc, taCenter|taBaseline)
	drawTextCenter(hdc, "×", x+w/2, y+h/2+4)
}

// drawTextCenter 居中绘制文字（GDI TextOutW 用 TA_CENTER/TA_BASELINE）
func drawTextCenter(hdc uintptr, s string, x, y int) {
	ptr, _ := syscall.UTF16PtrFromString(s)
	procTextOut.Call(hdc, uintptr(x), uintptr(y),
		uintptr(unsafe.Pointer(ptr)), uintptr(len([]rune(s))))
}
