# paste_tool

一个基于 Win32 的托盘粘贴工具。当前版本已经把核心粘贴流程和 Windows 平台接入层拆开，便于后续做跨平台版本。

## 提要

这工具本来是给我的前女友方便做 PTA、头歌 等平台做题搞的，最近想了想，把它整理和优化之后就开源了。

## 更新渠道

- 仓库地址：`https://github.com/Mai-xiyu/Paste-Tool`
- Release 列表：`https://github.com/Mai-xiyu/Paste-Tool/releases`
- 最新版本检查：`https://github.com/Mai-xiyu/Paste-Tool/releases/latest`
- 最新便携版直链：`https://github.com/Mai-xiyu/Paste-Tool/releases/latest/download/paste_tool-latest-windows-x64.exe`
- 最新安装包直链：`https://github.com/Mai-xiyu/Paste-Tool/releases/latest/download/paste_tool-installer-latest.exe`
- 程序托盘菜单里的“检查更新”会直接打开 latest release 页面。
- 程序托盘菜单还提供“关于”和“仓库主页”入口，方便直接查看版本和项目地址。
- 程序托盘菜单还支持自动下载 latest 便携版或 latest 安装包到 Downloads 目录。

## 安装新版本

1. 打开 latest release 页面。
2. 下载最新 release 附件中的可执行文件或安装包，或者直接使用程序托盘菜单自动下载。
3. 关闭旧版本程序后，用新文件覆盖或运行安装包完成更新。

## GitHub Actions

### 1. 每次推送构建

- 工作流文件：`.github/workflows/ci-build.yml`
- 触发时机：任意 push、Pull Request、手动触发。
- 产物位置：GitHub Actions 运行记录中的 artifact。
- 产物内容：
	- `paste_tool-v<version>-windows-x64.exe`
	- `paste_tool-installer-v<version>-windows-x64.exe`

### 2. 推送版本 Tag 发 Release

- 工作流文件：`.github/workflows/release.yml`
- 触发时机：推送形如 `v1.0.0` 的 tag，或手动触发。
- 校验规则：release tag 必须和 `app_metadata.h` 里的 `APP_VERSION` 一致，例如 `APP_VERSION = 0.1.0` 时只能发布 `v0.1.0`。
- 产物位置：
	- Actions 运行记录中的 artifact。
	- 对应 GitHub Release 下的附件。
- Release 资产命名：
	- `paste_tool-<tag>-windows-x64.exe`
	- `paste_tool-latest-windows-x64.exe`
	- `paste_tool-installer-<tag>-windows-x64.exe`
	- `paste_tool-installer-latest.exe`

### 3. 推荐发布流程

1. 更新 `app_metadata.h` 里的版本号。
2. 提交并推送到 `main`，确认 CI 构建正常。
3. 打 tag，例如：`git tag v0.1.0`。
4. 推送 tag：`git push origin v0.1.0`。
5. 等待 GitHub Actions 自动构建并把产物挂到 Release。

## 构建产物

- GitHub Actions 和 GitHub Release 都直接提供 `.exe` 文件下载，不再额外打包为 zip。
- 便携版 exe 适合直接下载覆盖使用。
- 安装包 exe 适合标准安装和后续升级。

## 文件结构

- `paste_tool.c`: Windows 程序入口。
- `platform_win32.c`: 托盘、热键、剪贴板、消息循环等 Win32 平台实现。
- `platform_win32.h`: Windows 平台入口声明。
- `app_core.c`: 平台无关的默认配置和文本粘贴流程。
- `app_core.h`: 核心层公开类型和接口。
- `installer/PasteTool.iss`: Inno Setup 安装包脚本。
- `scripts/build-installer.ps1`: 安装包构建脚本。
- `scripts/package-artifact.ps1`: 便携版可执行文件整理脚本。

## 构建

使用 GCC/MinGW：

```bash
gcc paste_tool.c platform_win32.c app_core.c -o paste_tool.exe -mwindows -lshell32 -lurlmon
```

## 后续跨平台建议

1. 保留 `app_core.*` 作为公共核心层。
2. 新增 `platform_linux.*` 或 `platform_macos.*` 实现热键、托盘和剪贴板。
3. 为不同平台提供各自的入口文件，复用相同的核心配置与粘贴策略。