# Paste Tool

Paste Tool 是一个 Go 实现的跨平台托盘粘贴工具。它通过模拟键盘逐字符输入文本，用于不可靠或不支持 `Ctrl+V` 的终端、远程桌面、VNC、在线代码编辑器和答题平台。

## 提要

这个工具本来是给我的前女友方便做 PTA、头歌等平台做题搞的，最近想了想，把它整理和优化之后开源了。

## 功能

- 跨平台核心：Windows、macOS、Linux X11。
- 明确边界：Linux Wayland 无 X11 `DISPLAY` 时返回 unsupported，不伪装成可用。
- CLI：`paste`、`doctor`、`config`、`update`、`version`。
- GUI：Fyne 托盘常驻和设置窗口。
- 默认热键：`Ctrl+Alt+V`。
- 默认粘贴参数：启动延迟 `3000ms`、字符间隔 `8ms`、批量 `50`、批间暂停 `20ms`。
- 更新检查：读取 GitHub latest release，支持下载便携版或安装包资产。

## 平台要求

- Windows：使用 `SendInput`。如果目标窗口权限级别高于 Paste Tool，Windows UIPI 可能阻止输入。
- macOS：使用 CoreGraphics 事件注入，需要 Accessibility/Input Monitoring 权限。
- Linux：X11 使用 XTest；Wayland 默认不允许通用全局输入注入，本项目只给出明确错误提示。

## 使用

启动托盘 GUI：

```bash
paste-tool
```

CLI 粘贴剪贴板文本：

```bash
paste-tool paste --source clipboard
```

从参数 dry-run 验证文本规范化：

```bash
paste-tool paste --source arg --text "hello" --dry-run
```

诊断当前平台：

```bash
paste-tool doctor
```

修改配置：

```bash
paste-tool config set hotkey Ctrl+Alt+V
paste-tool config set ui.language zh-CN
paste-tool config set paste.start_delay_ms 3000
paste-tool config get
```

检查和下载更新：

```bash
paste-tool update check
paste-tool update download portable
paste-tool update download installer
```

## 从源码构建

需要 Go。Linux 构建 GUI 和 X11 输入层时还需要桌面开发库，例如 Debian/Ubuntu：

```bash
sudo apt-get install gcc libgl1-mesa-dev xorg-dev libxkbcommon-dev libxtst-dev
```

构建：

```bash
go build -o dist/paste_tool ./cmd/paste-tool
```

Windows 发布版使用无控制台 GUI 入口，避免双击时弹出终端：

```bash
go build -ldflags "-H windowsgui" -o dist/paste_tool.exe ./cmd/paste-tool-gui
```

测试：

```bash
go test ./...
go vet ./...
```

## 配置文件

配置文件位于系统用户配置目录：

```bash
paste-tool config path
```

默认内容等价于：

```json
{
  "hotkey": {
    "modifiers": ["Ctrl", "Alt"],
    "key": "V"
  },
  "paste": {
    "start_delay_ms": 3000,
    "inter_key_delay_ms": 8,
    "batch_size": 50,
    "batch_pause_ms": 20
  },
  "update": {
    "repository": "Mai-xiyu/Paste-Tool"
  },
  "ui": {
    "language": "auto"
  }
}
```

`ui.language` 支持 `auto`、`zh-CN`、`en`。默认 `auto` 会跟随系统语言，无法识别时回退到英文。
