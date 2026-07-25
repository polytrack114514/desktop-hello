# 桌面按键实时显示（BongoCat 风格）设计文档

- 日期：2026-07-25
- 仓库：`polytrack114514/desktop-hello-` 子目录 `key-display/`
- 状态：已批准

## 1. 目标与范围

做一个 Windows 桌面小程序，在屏幕上以半透明置顶面板实时显示键盘和鼠标按键状态，类似 BongoCat。仅显示按键状态，不记录、不上传、不显示鼠标移动/滚轮。

### 需求确认

- 显示范围：键盘按键 + 鼠标按键（左/中/右）
- 显示风格：完整键盘示意图 + 鼠标示意图，按下高亮
- 面板背景：半透明黑底（70% 不透明）
- 位置：可拖动到任意位置，默认屏幕底部居中
- 高亮效果：按下变色（橙色）+ 轻微缩放（1.1 倍），松开 150ms 回弹
- 退出方式：面板右上角 × 关闭按钮
- 代码位置：`desktop-hello-` 仓库 `key-display/` 子目录

## 2. 技术选型

**方案 A（采纳）：纯 Go + Win32 API + GDI**

理由：
- 无外部依赖，Go 1.25 交叉编译已验证可行（参考 [main.go](file:///workspace/desktop-hello/main.go)）
- apt 源连不上、mingw 装不了，方案 B/C（walk/Fyne）不可行
- 产物体积小（~2MB），性能足够

## 3. 架构

### 3.1 模块划分

```
key-display/
├── main.go       # 入口，启动钩子 + 窗口
├── window.go     # 注册窗口类、消息循环、拖动、退出
├── keyhook.go    # 全局键盘/鼠标钩子注册与回调
├── draw.go       # GDI 绘制键盘 + 鼠标示意图
├── keymap.go     # VK 码 → 键盘位置/标签 映射表
└── go.mod
```

### 3.2 数据流

1. `keyhook` 用 `SetWindowsHookEx(WH_KEYBOARD_LL/WH_MOUSE_LL)` 注册全局钩子
2. 钩子回调（运行在系统注入线程）通过 `PostMessage(hwnd, WM_USER+1, ...)` 发事件到主线程
3. `window` 主消息循环处理 `WM_USER+1`，更新 `KeyState`
4. 调用 `InvalidateRect` → `WM_PAINT` → `draw` 重绘

## 4. 键盘与鼠标布局

### 4.1 键盘布局（QWERTY 美式，6 行）

```
Esc  F1..F12
` 1..0 - = Backspace
Tab Q W E R T Y U I O P [ ] \
Caps A S D F G H J K L ; ' Enter
Shift Z X C V B N M , . / Shift
Ctrl Win Alt Space Alt Win Menu Ctrl
```

### 4.2 鼠标示意图（键盘右侧）

椭圆体鼠标轮廓 + 左右键分区 + 中键椭圆。滚轮仅静态画图，不响应。

### 4.3 尺寸

- 整体面板：720×220px
- 单个标准键：40×40px，间距 4px
- 修饰键按实际宽度拉伸
- 鼠标示意：100×130px，紧贴键盘右侧

### 4.4 配色

- 默认键：深灰底 `RGB(60,60,60)` + 浅灰边框
- 按下键：橙色填充 `RGB(255,165,0)` + 白色文字
- 面板背景：黑色 70% 不透明

## 5. 窗口与交互

### 5.1 窗口样式

```
WS_POPUP | WS_VISIBLE | WS_CLIPCHILDREN
WS_EX_LAYERED      # 半透明
WS_EX_TOPMOST      # 置顶
WS_EX_TRANSPARENT  # 鼠标点击穿透（×按钮区域例外）
WS_EX_TOOLWINDOW   # 不显示在任务栏
```

### 5.2 半透明

`SetLayeredWindowAttributes(hwnd, 0, 180, LWA_ALPHA)` → 70% 不透明。

### 5.3 拖动

`WM_NCHITTEST` 在非 ×按钮区域返回 `HTCAPTION`，系统自动处理拖动。

### 5.4 退出

- × 按钮：30×30px，`RGB(200,50,50)`，位置面板右上角
- 该区域 `WM_NCHITTEST` 返回 `HTCLIENT`，自处理 `WM_LBUTTONDOWN` 发 `WM_CLOSE`

### 5.5 初始位置

`GetSystemMetrics(SM_CXSCREEN/SM_CYSCREEN)`，计算 `(screenW-720)/2, screenH-240`。

## 6. 钩子与状态管理

### 6.1 钩子

- `WH_KEYBOARD_LL` (13) + `WH_MOUSE_LL` (14)
- `SetWindowsHookEx` 第 4 参数 0 = 全桌面
- 回调只读 `lParam`，不写；事件经 `PostMessage` 跨线程传递

### 6.2 鼠标按键判定（wParam）

- `WM_LBUTTONDOWN/UP` → 左键
- `WM_RBUTTONDOWN/UP` → 右键
- `WM_MBUTTONDOWN/UP` → 中键

### 6.3 KeyState

```go
type KeyState struct {
    keys  map[uint8]bool      // 当前按下的 VK 码
    anim  map[uint8]time.Time // 每键动画开始时间
    mouse struct{ left, middle, right bool }
}
```

### 6.4 生命周期

退出时 `UnhookWindowsHookEx` 释放两个钩子。

## 7. GDI 绘制

### 7.1 流程

```
BeginPaint
  ├─ CreateCompatibleDC + bitmap（双缓冲）
  ├─ 填充半透明黑底
  ├─ 遍历 keymap 画每个键（RoundRect 8px 圆角）
  ├─ 画鼠标示意（椭圆 + 左右半区 + 中键椭圆）
  ├─ 画 × 按钮
  └─ BitBlt 内存 DC → 屏幕 DC
EndPaint
```

### 7.2 缩放与颜色插值

- 按下时 `scale = 1.1`，以键中心为锚点重算左上角
- 松开 150ms 内：`t = time.Since(anim[vk]) / 150ms`，`scale = 1.1 - 0.1*t`
- 颜色：(255,165,0) → (60,60,60)，按 t 线性插值

### 7.3 字体

`CreateFont(14, ..., "Segoe UI")` 加粗，默认浅灰 `RGB(220,220,220)`，按下白色。

### 7.4 VK 标签表

| VK | 标签 | VK | 标签 |
|----|------|----|------|
| 0x10 | Shift | 0x20 | Space |
| 0x11 | Ctrl | 0x0D | Enter |
| 0x12 | Alt | 0x1B | Esc |
| 0x5B | Win | 0x09 | Tab |
| 0x14 | Caps | 0x08 | ← |
| 0xA0 | LShift | 0xA1 | RShift |
| 0xA2 | LCtrl | 0xA3 | RCtrl |
| 0xA4 | LAlt | 0xA5 | RAlt |

## 8. 重绘节流

- 单次事件只 `InvalidateRect` 一次
- 动画期间 `SetTimer(hwnd, 1, 16, nil)` 启动 60fps 重绘
- 所有动画结束 → `KillTimer`，静态时 0 CPU

## 9. 编译与构建

```
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -ldflags="-H windowsgui" \
  -o key-display.exe .
```

产物为 PE32+ GUI 子系统 exe，Windows 双击运行，无控制台黑窗。

## 10. 验收标准

1. 在 Windows 10/11 双击 `key-display.exe` 能启动
2. 面板半透明置顶，默认底部居中
3. 按键盘任意键，对应键高亮橙色 + 轻微放大
4. 松开后 150ms 内回弹至原状
5. 按鼠标左/中/右键，鼠标示意图对应区域高亮
6. 可拖动面板到任意位置
7. 点 × 按钮程序退出
8. 静止时 CPU 接近 0

## 11. 已知限制

- Linux 上无法运行测试，只能交叉编译验证 PE 格式
- `WS_EX_TRANSPARENT` 会让面板大部分区域点击穿透，但 ×按钮区域例外
- 钩子覆盖当前桌面所有窗口，不能仅对某应用生效
