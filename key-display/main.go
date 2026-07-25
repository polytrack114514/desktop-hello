package main

func main() {
	// 初始化字体
	initFonts()

	// 创建并显示主窗口
	hwnd := createMainWindow()
	if hwnd == 0 {
		return
	}
	procShowWindow.Call(hwnd, 5) // SW_SHOW=5

	// 进入消息循环
	messageLoop()
}
