<p align="center">
  <img src="key-display/app.ico.png" width="128" height="128" alt="Logo">
</p>

<h1 align="center">Desktop Hello</h1>

<p align="center">
  <a href="https://github.com/polytrack114514/desktop-hello/releases">
    <img src="https://img.shields.io/github/v/release/polytrack114514/desktop-hello?style=flat-square&label=version" alt="Release">
  </a>
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=flat-square&logo=go" alt="Go">
  <img src="https://img.shields.io/badge/Platform-Windows-0078D6?style=flat-square&logo=windows" alt="Platform">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  <a href="https://github.com/polytrack114514/desktop-hello/releases">
    <img src="https://img.shields.io/github/downloads/polytrack114514/desktop-hello/total?style=flat-square&color=blue" alt="Downloads">
  </a>
  <a href="https://github.com/polytrack114514/desktop-hello/stargazers">
    <img src="https://img.shields.io/github/stars/polytrack114514/desktop-hello?style=flat-square&color=yellow" alt="Stars">
  </a>
</p>

<p align="center">
  <a href="#中文">中文</a> | <a href="#english">English</a>
</p>

---

## 中文

一款轻量的 Windows 桌面按键显示工具，适用于游戏直播、教程录制、按键展示等场景。

### 功能

- 实时显示键盘按键状态（按下/松开）
- 鼠标左键/右键点击显示
- 暗色 / 亮色模式切换
- 透明度调节（30% - 100%）
- 面板大小调节（50% - 100%）
- 中英文语言切换
- 面板显示当前时间（年月日时分秒）
- 精简按键布局，去掉游戏不常用的符号键和右侧 Win/Menu/Ctrl
- 启动不抢占焦点，不影响系统输入法状态
- 系统托盘常驻，右键菜单操作
- 设置自动保存，重启后恢复
- 暗色卡片风格设置界面

### 下载

前往 [Releases](https://github.com/polytrack114514/desktop-hello/releases) 下载最新版本 `key-display.exe`，直接运行即可。

### 使用方法

1. 下载 `key-display.exe` 并运行
2. 程序自动驻留系统托盘
3. 右键托盘图标，选择「设置」进行个性化配置
4. 在设置中调整透明度、面板大小、主题模式和语言

### 快捷菜单

| 菜单项 | 功能 |
|--------|------|
| 设置 | 打开设置窗口 |
| GitHub 仓库 | 跳转 GitHub 仓库 |
| 退出 | 关闭程序 |

### 技术栈

- Go + Windows API (user32.dll / gdi32.dll)
- 纯 Win32 原生窗口，无外部依赖

---

## English

A lightweight Windows desktop key display tool for game streaming, tutorial recording, and key visualization.

### Features

- Real-time keyboard key state display (press/release)
- Mouse left/right click display
- Dark / Light mode toggle
- Opacity adjustment (30% - 100%)
- Panel scale adjustment (50% - 100%)
- Chinese / English language switch
- Live clock display (date and time)
- Simplified key layout — removed uncommon symbol keys and right-side Win/Menu/Ctrl
- No focus stealing on launch — does not affect system IME state
- System tray with right-click context menu
- Auto-save settings, restored on restart
- Dark card-style settings UI

### Download

Go to [Releases](https://github.com/polytrack114514/desktop-hello/releases) and download the latest `key-display.exe`. Run it directly.

### Usage

1. Download and run `key-display.exe`
2. The app resides in the system tray
3. Right-click the tray icon and select **Settings** to configure
4. Adjust opacity, panel scale, theme mode, and language in settings

### Context Menu

| Menu Item | Action |
|-----------|--------|
| Settings | Open settings window |
| GitHub Repo | Open GitHub repository |
| Exit | Quit the application |

### Tech Stack

- Go + Windows API (user32.dll / gdi32.dll)
- Pure Win32 native window, no external dependencies
