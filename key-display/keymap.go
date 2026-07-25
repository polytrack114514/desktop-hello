package main

// Windows 虚拟键码常量
const (
	vkEsc       = 0x1B
	vkTab       = 0x09
	vkCapsLock  = 0x14
	vkShift     = 0x10
	vkCtrl      = 0x11
	vkAlt       = 0x12
	vkEnter     = 0x0D
	vkBackspace = 0x08
	vkSpace     = 0x20
	vkLWin      = 0x5B
	vkRWin      = 0x5C
	vkMenu      = 0x5D
	vkLShift    = 0xA0
	vkRShift    = 0xA1
	vkLCtrl     = 0xA2
	vkRCtrl     = 0xA3
	vkLAlt      = 0xA4
	vkRAlt      = 0xA5

	vkOem3  = 0xC0 // `
	vkOemMinus = 0xBD // -
	vkOemPlus  = 0xBB // =
	vkOem4  = 0xDB // [
	vkOem6  = 0xDD // ]
	vkOem5  = 0xDC // \
	vkOem1  = 0xBA // ;
	vkOem7  = 0xDE // '
	vkOemComma  = 0xBC // ,
	vkOemPeriod = 0xBE // .
	vkOem2  = 0xBF // /

	vkF1  = 0x70
	vkF12 = 0x7B
)

// KeyDef 键盘上单个键的位置与标签
type KeyDef struct {
	VK   uint16
	X, Y int    // 左上角坐标（px）
	W, H int    // 宽高（px）
	Label string // 显示文本
}

// keyMap 键盘所有键定义，面板原点 (0,0) 即面板左上角
// 面板宽 720 高 220，键 40x40，间距 4
var keyMap = buildKeyMap()

const (
	keySize = 40
	keyGap  = 4
	originX = 8
	originY = 8
	panelW  = 720
	panelH  = 220
	mouseX  = panelW - 8 - 100 // 鼠标示意左上角 X
	mouseY  = 8
	mouseW  = 100
	mouseH  = 170
)

func buildKeyMap() []KeyDef {
	var keys []KeyDef
	// 行 1：Esc + F1..F12
	x := originX
	keys = append(keys, KeyDef{vkEsc, x, originY, keySize, keySize, "Esc"})
	x += keySize + keyGap
	for i := 0; i < 12; i++ {
		var lbl string
		if i < 9 {
			lbl = fLabel(i)
		} else {
			lbl = fLabel10Plus(i)
		}
		keys = append(keys, KeyDef{uint16(vkF1) + uint16(i), x, originY, 36, keySize, lbl})
		x += 36 + keyGap
	}

	// 行 2：` 1..0 - = Backspace
	y := originY + keySize + keyGap
	x = originX
	row2 := []struct {
		vk    uint16
		label string
		w     int
	}{
		{vkOem3, "`", keySize},
		{0x30, "1", keySize}, {0x31, "2", keySize}, {0x32, "3", keySize},
		{0x33, "4", keySize}, {0x34, "5", keySize}, {0x35, "6", keySize},
		{0x36, "7", keySize}, {0x37, "8", keySize}, {0x38, "9", keySize},
		{0x39, "0", keySize},
		{vkOemMinus, "-", keySize}, {vkOemPlus, "=", keySize},
		{vkBackspace, "←", 88},
	}
	for _, k := range row2 {
		keys = append(keys, KeyDef{k.vk, x, y, k.w, keySize, k.label})
		x += k.w + keyGap
	}

	// 行 3：Tab Q..P [ ] \
	y = originY + 2*(keySize+keyGap)
	x = originX
	row3 := []struct {
		vk    uint16
		label string
		w     int
	}{
		{vkTab, "Tab", 64},
		{0x51, "Q", keySize}, {0x57, "W", keySize}, {0x45, "E", keySize},
		{0x52, "R", keySize}, {0x54, "T", keySize}, {0x59, "Y", keySize},
		{0x55, "U", keySize}, {0x49, "I", keySize}, {0x4F, "O", keySize},
		{0x50, "P", keySize},
		{vkOem4, "[", keySize}, {vkOem6, "]", keySize},
		{vkOem5, "\\", 64},
	}
	for _, k := range row3 {
		keys = append(keys, KeyDef{k.vk, x, y, k.w, keySize, k.label})
		x += k.w + keyGap
	}

	// 行 4：Caps A..L ; ' Enter
	y = originY + 3*(keySize+keyGap)
	x = originX
	row4 := []struct {
		vk    uint16
		label string
		w     int
	}{
		{vkCapsLock, "Caps", 76},
		{0x41, "A", keySize}, {0x53, "S", keySize}, {0x44, "D", keySize},
		{0x46, "F", keySize}, {0x47, "G", keySize}, {0x48, "H", keySize},
		{0x4A, "J", keySize}, {0x4B, "K", keySize}, {0x4C, "L", keySize},
		{vkOem1, ";", keySize}, {vkOem7, "'", keySize},
		{vkEnter, "Enter", 84},
	}
	for _, k := range row4 {
		keys = append(keys, KeyDef{k.vk, x, y, k.w, keySize, k.label})
		x += k.w + keyGap
	}

	// 行 5：Shift Z..M , . / Shift
	y = originY + 4*(keySize+keyGap)
	x = originX
	row5 := []struct {
		vk    uint16
		label string
		w     int
	}{
		{vkLShift, "Shift", 88},
		{0x5A, "Z", keySize}, {0x58, "X", keySize}, {0x43, "C", keySize},
		{0x56, "V", keySize}, {0x42, "B", keySize}, {0x4E, "N", keySize},
		{0x4D, "M", keySize},
		{vkOemComma, ",", keySize}, {vkOemPeriod, ".", keySize},
		{vkOem2, "/", keySize},
		{vkRShift, "Shift", 88},
	}
	for _, k := range row5 {
		keys = append(keys, KeyDef{k.vk, x, y, k.w, keySize, k.label})
		x += k.w + keyGap
	}

	// 行 6：Ctrl Win Alt Space Alt Win Menu Ctrl
	y = originY + 5*(keySize+keyGap)
	x = originX
	row6 := []struct {
		vk    uint16
		label string
		w     int
	}{
		{vkLCtrl, "Ctrl", 52},
		{vkLWin, "Win", 52},
		{vkLAlt, "Alt", 52},
		{vkSpace, "Space", 268},
		{vkRAlt, "Alt", 52},
		{vkRWin, "Win", 52},
		{vkMenu, "≡", 52},
		{vkRCtrl, "Ctrl", 52},
	}
	for _, k := range row6 {
		keys = append(keys, KeyDef{k.vk, x, y, k.w, keySize, k.label})
		x += k.w + keyGap
	}

	return keys
}

func fLabel(i int) string {
	return "F" + string(rune('1'+i)) // i ∈ [0,8] -> F1..F9
}

// fLabel10Plus 返回 F10/F11/F12（避免 '1'+i 越界）
func fLabel10Plus(i int) string {
	return []string{"F10", "F11", "F12"}[i-9]
}
