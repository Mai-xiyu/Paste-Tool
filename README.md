# Paste Tool

一个轻量级的跨平台托盘粘贴工具，通过模拟键盘输入逐字符粘贴文本，绕过不支持 Ctrl+V 的平台限制（如 PTA、头歌等在线答题平台的代码编辑器）。

基于 Qt6 构建，支持 Windows（macOS / Linux 后续支持中）。

## 提要

这工具本来是给我的前女友方便做 PTA、头歌 等平台做题搞的，最近想了想，把它整理和优化之后就开源了。

## 功能特性

- **逐字符模拟输入**：兼容终端（PuTTY、SSH、Windows Terminal）和浏览器在线编辑器
- **系统托盘运行**：不占桌面空间，右键菜单操作
- **自定义热键**：默认 Ctrl+Alt+V，可在托盘菜单「更改热键」中自由配置（Ctrl/Alt/Shift/Win + A-Z/0-9/F1-F12）
- **检查更新**：托盘菜单一键检查 GitHub 最新版本，自动比对版本号提示更新
- **一键下载**：支持直接下载最新便携版或安装包到 Downloads 目录
- **跨平台架构**：Qt6 + CMake，核心粘贴算法平台无关

## 下载安装

### 便携版（推荐）

直接下载 exe，双击运行即可：

- [最新便携版下载](https://github.com/Mai-xiyu/Paste-Tool/releases/latest/download/paste_tool-latest-windows-x64.exe)

### 安装包

标准安装，支持开始菜单和桌面快捷方式：

- [最新安装包下载](https://github.com/Mai-xiyu/Paste-Tool/releases/latest/download/paste_tool-installer-latest.exe)

### 历史版本

- [所有 Release](https://github.com/Mai-xiyu/Paste-Tool/releases)

## 使用方法

1. **复制**：先复制你要粘贴的代码或文本（Ctrl+C）
2. **触发**：按下快捷键（默认 Ctrl+Alt+V）
3. **准备**：听到提示音后，在 3 秒内切换到目标输入框
4. **粘贴**：程序自动逐字符模拟键盘输入，完成后热键自动恢复

### 托盘菜单

右键系统托盘图标可使用以下功能：

| 菜单项 | 说明 |
|--------|------|
| 关于 | 查看版本信息和项目链接 |
| 使用说明 | 查看快捷键和使用帮助 |
| 更改热键 | 自定义快捷键组合 |
| 检查更新 | 查询 GitHub 最新版本 |
| 下载最新便携版 | 自动下载到 Downloads 目录 |
| 下载最新安装包 | 自动下载并可选启动安装 |
| 仓库主页 | 打开 GitHub 项目页面 |
| 退出 | 关闭程序 |

## 更新方式

- **程序内检查**：托盘菜单 →「检查更新」，有新版本会提示并可跳转下载
- **程序内下载**：托盘菜单 →「下载最新便携版」或「下载最新安装包」，直接下载到 Downloads 目录
- **手动更新**：前往 [Release 页面](https://github.com/Mai-xiyu/Paste-Tool/releases/latest) 下载最新版本

## 从源码构建

需要 Qt6 和 CMake：

```bash
cmake -B build -DCMAKE_BUILD_TYPE=Release
cmake --build build --config Release
```

Windows 上还需运行 `windeployqt` 部署 Qt 运行库：

```bash
windeployqt --release build/src/Release/paste_tool.exe
```

## 许可

开源项目，仓库地址：https://github.com/Mai-xiyu/Paste-Tool
