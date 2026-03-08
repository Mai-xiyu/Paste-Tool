# paste_tool

一个基于 Win32 的托盘粘贴工具。当前版本已经把核心粘贴流程和 Windows 平台接入层拆开，便于后续做跨平台版本。

## 更新渠道

- 仓库地址：`https://github.com/Mai-xiyu/Paste-Tool`
- Release 列表：`https://github.com/Mai-xiyu/Paste-Tool/releases`
- 最新版本检查：`https://github.com/Mai-xiyu/Paste-Tool/releases/latest`
- 程序托盘菜单里的“检查更新”会直接打开 latest release 页面。
- 后续发布新版本时，建议把可执行文件或安装包作为 GitHub release 附件上传。

## 安装新版本

1. 打开 latest release 页面。
2. 下载最新 release 附件中的可执行文件或安装包。
3. 关闭旧版本程序后，用新文件覆盖或运行安装包完成更新。

## 文件结构

- `paste_tool.c`: Windows 程序入口。
- `platform_win32.c`: 托盘、热键、剪贴板、消息循环等 Win32 平台实现。
- `platform_win32.h`: Windows 平台入口声明。
- `app_core.c`: 平台无关的默认配置和文本粘贴流程。
- `app_core.h`: 核心层公开类型和接口。

## 构建

使用 GCC/MinGW：

```bash
gcc paste_tool.c platform_win32.c app_core.c -o paste_tool.exe -mwindows -lshell32
```

## 后续跨平台建议

1. 保留 `app_core.*` 作为公共核心层。
2. 新增 `platform_linux.*` 或 `platform_macos.*` 实现热键、托盘和剪贴板。
3. 为不同平台提供各自的入口文件，复用相同的核心配置与粘贴策略。