#include "platform_win32.h"

#include "app_core.h"
#include "app_metadata.h"

#include <shellapi.h>
#include <stdlib.h>
#include <strsafe.h>
#include <urlmon.h>
#include <winhttp.h>

#define WM_APP_TRAYICON (WM_USER + 1)
#define WM_APP_PASTE_DONE (WM_USER + 2)

#define APP_TRAY_ICON_ID 1
#define APP_HOTKEY_ID 100

#define IDM_ABOUT 1999
#define IDM_CHECK_UPDATE 2000
#define IDM_OPEN_REPOSITORY 2001
#define IDM_DOWNLOAD_PORTABLE 2002
#define IDM_DOWNLOAD_INSTALLER 2003
#define IDM_HELP 2004
#define IDM_EXIT 2005
#define IDM_CHANGE_HOTKEY 2006

#define IDC_CHECK_CTRL  3001
#define IDC_CHECK_ALT   3002
#define IDC_CHECK_SHIFT 3003
#define IDC_CHECK_WIN   3004
#define IDC_COMBO_KEY   3005
#define IDC_BTN_OK      3006
#define IDC_BTN_CANCEL  3007

#define HOTKEY_DIALOG_CLASS L"PasteToolHotkeyDialog"

#define APP_WINDOW_CLASS_NAME L"AutoPasteToolClass"
#define APP_WINDOW_TITLE L"PasteTool"

#define REG_PATH L"Software\\AutoPasteTool"
#define REG_KEY_MODIFIERS L"KeyModifier"
#define REG_KEY_VIRTUAL_KEY L"KeyVirtual"

typedef struct Win32AppState {
    HINSTANCE instance;
    HWND messageWindow;
    LONG isPasting;
    NOTIFYICONDATAW trayIcon;
    AppConfig config;
} Win32AppState;

static Win32AppState g_app;

static LRESULT CALLBACK AppWindowProc(HWND windowHandle, UINT message, WPARAM wParam, LPARAM lParam);

static void AppShowError(const wchar_t* message);
static void AppShowAbout(void);
static void AppShowHelp(void);
static void AppCheckForUpdates(void);
static void AppOpenRepository(void);
static void AppDownloadLatestPortable(void);
static void AppDownloadLatestInstaller(void);
static void AppLoadConfigFromRegistry(void);
static BOOL AppRegisterWindowClass(HINSTANCE instance);
static BOOL AppRegisterHotkey(HWND windowHandle);
static BOOL AppUnregisterHotkey(HWND windowHandle);
static void AppDrainPendingHotkeyMessages(HWND windowHandle);
static void AppInitializeTrayIcon(HWND windowHandle);
static void AppUpdateTrayTooltip(void);
static void AppRemoveTrayIcon(void);
static void AppShowTrayMenu(HWND windowHandle);
static void AppHandleTrayCommand(HWND windowHandle, UINT commandId);
static wchar_t* AppReadClipboardText(void);
static void AppOpenUrl(const wchar_t* url);
static BOOL AppGetDownloadsDirectory(wchar_t* downloadsDirectory, size_t capacity);
static void AppOpenDirectory(const wchar_t* directoryPath);
static void AppDownloadReleaseAsset(const wchar_t* downloadUrl, const wchar_t* targetFileName, BOOL launchAfterDownload);
static void AppSleepMs(void* userData, uint32_t milliseconds);
static void AppNotifyPasteStart(void* userData);
static void AppNotifyPasteError(void* userData);
static void AppSendUnicodeCharacter(void* userData, wchar_t character);
static DWORD WINAPI AppPasteThreadProc(LPVOID parameter);
static BOOL AppStartPasteOperation(HWND windowHandle);
static void AppFinishPasteOperation(HWND windowHandle);
static void AppBuildModifierString(uint32_t modifiers, wchar_t* buffer, size_t capacity);
static const wchar_t* AppGetKeyName(uint32_t virtualKey);
static void AppSaveHotkeyToRegistry(void);
static BOOL AppShowChangeHotkeyDialog(void);
static BOOL AppFetchLatestVersionTag(wchar_t* tagBuffer, size_t tagBufferSize);
static int AppParseVersionComponent(const wchar_t** str);
static int AppCompareVersionStrings(const wchar_t* current, const wchar_t* latest);

static void AppShowError(const wchar_t* message) {
    MessageBoxW(NULL, message, L"错误", MB_ICONERROR);
}

static void AppShowAbout(void) {
    wchar_t message[512];

    StringCchPrintfW(
        message,
        ARRAYSIZE(message),
        L"%ls\n版本：%ls\n\n"
        L"仓库主页：\n%ls\n\n"
        L"更新检查：\n%ls\n\n"
        L"便携版直链：\n%ls\n\n"
        L"安装包直链：\n%ls\n\n"
        L"新版本建议从 GitHub Release 页面下载安装。",
        APP_NAME,
        APP_VERSION,
        APP_REPOSITORY_URL,
        APP_LATEST_RELEASE_URL,
        APP_LATEST_PORTABLE_DOWNLOAD_URL,
        APP_LATEST_INSTALLER_DOWNLOAD_URL
    );

    MessageBoxW(NULL, message, L"关于", MB_ICONINFORMATION);
}

static void AppShowHelp(void) {
    wchar_t message[512];
    wchar_t modifierText[64];

    AppBuildModifierString(g_app.config.hotkeyModifiers, modifierText, ARRAYSIZE(modifierText));

    StringCchPrintfW(
        message,
        ARRAYSIZE(message),
        L"【%ls %ls 使用说明】\n\n"
        L"1. 复制：先复制你要输入的代码或文本。\n"
        L"2. 触发：按下快捷键 %ls%ls。\n"
        L"3. 准备：听到提示音后，你有 %lu 秒切到目标输入框。\n"
        L"4. 粘贴：程序会自动模拟键盘输入。\n\n"
        L"注意事项：\n"
        L"- 输入期间会暂时禁用热键，避免重复触发。\n"
        L"- 程序在系统托盘运行，右键图标可查看帮助或退出。\n"
        L"- 可通过右键菜单「更改热键」自定义快捷键组合。\n"
        L"- 检查更新会查询 GitHub 最新版本并提示下载。\n"
        L"- 可直接下载 latest 便携版 exe 或安装包 exe。\n"
        L"- 核心粘贴逻辑已拆到 app_core.c，便于后续做跨平台版本。",
        APP_NAME,
        APP_VERSION,
        modifierText,
        AppGetKeyName(g_app.config.hotkeyVirtualKey),
        (unsigned long)(g_app.config.startDelayMs / 1000U)
    );

    MessageBoxW(NULL, message, L"帮助", MB_ICONINFORMATION);
}

static void AppOpenUrl(const wchar_t* url) {
    HINSTANCE result = ShellExecuteW(NULL, L"open", url, NULL, NULL, SW_SHOWNORMAL);

    if ((INT_PTR)result <= 32) {
        AppShowError(L"无法打开链接，请稍后手动访问 GitHub release 页面。");
    }
}

static int AppParseVersionComponent(const wchar_t** str) {
    int value = 0;

    while (**str >= L'0' && **str <= L'9') {
        value = value * 10 + (**str - L'0');
        (*str)++;
    }

    if (**str == L'.') {
        (*str)++;
    }

    return value;
}

static int AppCompareVersionStrings(const wchar_t* current, const wchar_t* latest) {
    int curMajor, curMinor, curPatch;
    int latMajor, latMinor, latPatch;

    if (*current == L'v' || *current == L'V') current++;
    if (*latest == L'v' || *latest == L'V') latest++;

    curMajor = AppParseVersionComponent(&current);
    curMinor = AppParseVersionComponent(&current);
    curPatch = AppParseVersionComponent(&current);

    latMajor = AppParseVersionComponent(&latest);
    latMinor = AppParseVersionComponent(&latest);
    latPatch = AppParseVersionComponent(&latest);

    if (latMajor != curMajor) return latMajor - curMajor;
    if (latMinor != curMinor) return latMinor - curMinor;
    return latPatch - curPatch;
}

static BOOL AppFetchLatestVersionTag(wchar_t* tagBuffer, size_t tagBufferSize) {
    HINTERNET session = NULL;
    HINTERNET connection = NULL;
    HINTERNET request = NULL;
    char responseBuffer[4096];
    DWORD bytesRead;
    DWORD totalRead = 0;
    char* tagStart;
    char* tagEnd;
    BOOL success = FALSE;
    int i;

    session = WinHttpOpen(
        L"PasteTool",
        WINHTTP_ACCESS_TYPE_DEFAULT_PROXY,
        WINHTTP_NO_PROXY_NAME,
        WINHTTP_NO_PROXY_BYPASS,
        0
    );
    if (!session) goto cleanup;

    WinHttpSetTimeouts(session, 10000, 10000, 10000, 10000);

    connection = WinHttpConnect(session, L"api.github.com", INTERNET_DEFAULT_HTTPS_PORT, 0);
    if (!connection) goto cleanup;

    request = WinHttpOpenRequest(
        connection,
        L"GET",
        L"/repos/Mai-xiyu/Paste-Tool/releases/latest",
        NULL,
        WINHTTP_NO_REFERER,
        WINHTTP_DEFAULT_ACCEPT_TYPES,
        WINHTTP_FLAG_SECURE
    );
    if (!request) goto cleanup;

    WinHttpAddRequestHeaders(request, L"User-Agent: PasteTool\r\n", (ULONG)-1, WINHTTP_ADDREQ_FLAG_ADD);

    if (!WinHttpSendRequest(request, WINHTTP_NO_ADDITIONAL_HEADERS, 0, WINHTTP_NO_REQUEST_DATA, 0, 0, 0)) goto cleanup;
    if (!WinHttpReceiveResponse(request, NULL)) goto cleanup;

    while (WinHttpReadData(request, responseBuffer + totalRead,
           (DWORD)(sizeof(responseBuffer) - totalRead - 1), &bytesRead) && bytesRead > 0) {
        totalRead += bytesRead;
        if (totalRead >= sizeof(responseBuffer) - 1) break;
    }
    responseBuffer[totalRead] = '\0';

    tagStart = strstr(responseBuffer, "\"tag_name\"");
    if (!tagStart) goto cleanup;
    tagStart = strchr(tagStart + 10, '"');
    if (!tagStart) goto cleanup;
    tagStart++;
    tagEnd = strchr(tagStart, '"');
    if (!tagEnd || (size_t)(tagEnd - tagStart) >= tagBufferSize) goto cleanup;

    for (i = 0; tagStart + i < tagEnd; i++) {
        tagBuffer[i] = (wchar_t)(unsigned char)tagStart[i];
    }
    tagBuffer[i] = L'\0';
    success = TRUE;

cleanup:
    if (request) WinHttpCloseHandle(request);
    if (connection) WinHttpCloseHandle(connection);
    if (session) WinHttpCloseHandle(session);
    return success;
}

static void AppCheckForUpdates(void) {
    wchar_t latestTag[64];
    wchar_t message[512];
    int answer;

    if (!AppFetchLatestVersionTag(latestTag, ARRAYSIZE(latestTag))) {
        answer = MessageBoxW(NULL,
            L"无法获取最新版本信息，请检查网络连接。\n\n是否手动打开 GitHub Release 页面？",
            L"检查更新", MB_ICONWARNING | MB_YESNO);
        if (answer == IDYES) {
            AppOpenUrl(APP_LATEST_RELEASE_URL);
        }
        return;
    }

    if (AppCompareVersionStrings(APP_VERSION, latestTag) > 0) {
        StringCchPrintfW(message, ARRAYSIZE(message),
            L"发现新版本 %ls！\n当前版本：%ls\n\n是否打开下载页面？",
            latestTag, APP_VERSION);
        answer = MessageBoxW(NULL, message, L"检查更新", MB_ICONINFORMATION | MB_YESNO);
        if (answer == IDYES) {
            AppOpenUrl(APP_LATEST_RELEASE_URL);
        }
    } else {
        StringCchPrintfW(message, ARRAYSIZE(message),
            L"当前版本 %ls 已是最新版本。", APP_VERSION);
        MessageBoxW(NULL, message, L"检查更新", MB_ICONINFORMATION);
    }
}

static void AppOpenRepository(void) {
    AppOpenUrl(APP_REPOSITORY_URL);
}

static BOOL AppGetDownloadsDirectory(wchar_t* downloadsDirectory, size_t capacity) {
    DWORD length = GetEnvironmentVariableW(L"USERPROFILE", downloadsDirectory, (DWORD)capacity);

    if (length == 0 || length >= capacity) {
        return FALSE;
    }

    if (FAILED(StringCchCatW(downloadsDirectory, capacity, L"\\Downloads"))) {
        return FALSE;
    }

    CreateDirectoryW(downloadsDirectory, NULL);
    return TRUE;
}

static void AppOpenDirectory(const wchar_t* directoryPath) {
    HINSTANCE result = ShellExecuteW(NULL, L"open", directoryPath, NULL, NULL, SW_SHOWNORMAL);

    if ((INT_PTR)result <= 32) {
        AppShowError(L"无法打开下载目录，请手动前往 Downloads 文件夹查看。");
    }
}

static void AppDownloadReleaseAsset(const wchar_t* downloadUrl, const wchar_t* targetFileName, BOOL launchAfterDownload) {
    wchar_t downloadsDirectory[MAX_PATH];
    wchar_t outputPath[MAX_PATH];
    wchar_t message[1024];
    HRESULT result;
    int answer;

    if (!AppGetDownloadsDirectory(downloadsDirectory, ARRAYSIZE(downloadsDirectory))) {
        AppShowError(L"无法定位 Downloads 目录，下载失败。");
        return;
    }

    if (FAILED(StringCchPrintfW(outputPath, ARRAYSIZE(outputPath), L"%ls\\%ls", downloadsDirectory, targetFileName))) {
        AppShowError(L"下载路径过长，无法保存文件。");
        return;
    }

    result = URLDownloadToFileW(NULL, downloadUrl, outputPath, 0, NULL);
    if (FAILED(result)) {
        AppShowError(L"自动下载失败，请检查网络后重试，或手动打开 latest release 页面下载。");
        return;
    }

    if (launchAfterDownload) {
        if (FAILED(StringCchPrintfW(
            message,
            ARRAYSIZE(message),
            L"安装包已下载到：\n%ls\n\n是否现在启动安装包？",
            outputPath
        ))) {
            AppShowError(L"安装包已下载完成，但提示信息生成失败。");
            return;
        }

        answer = MessageBoxW(NULL, message, L"下载完成", MB_ICONINFORMATION | MB_YESNO);
        if (answer == IDYES) {
            AppOpenUrl(outputPath);
        } else {
            AppOpenDirectory(downloadsDirectory);
        }
        return;
    }

    if (FAILED(StringCchPrintfW(
        message,
        ARRAYSIZE(message),
        L"最新便携版已下载到：\n%ls\n\n是否打开下载目录？",
        outputPath
    ))) {
        AppShowError(L"便携版已下载完成，但提示信息生成失败。");
        return;
    }

    answer = MessageBoxW(NULL, message, L"下载完成", MB_ICONINFORMATION | MB_YESNO);
    if (answer == IDYES) {
        AppOpenDirectory(downloadsDirectory);
    }
}

static void AppDownloadLatestPortable(void) {
    AppDownloadReleaseAsset(
        APP_LATEST_PORTABLE_DOWNLOAD_URL,
        L"paste_tool-latest-windows-x64.exe",
        FALSE
    );
}

static void AppDownloadLatestInstaller(void) {
    AppDownloadReleaseAsset(
        APP_LATEST_INSTALLER_DOWNLOAD_URL,
        L"paste_tool-installer-latest.exe",
        TRUE
    );
}

static void AppLoadConfigFromRegistry(void) {
    HKEY registryKey;
    DWORD disposition;
    DWORD dataSize;

    AppConfigInitDefaults(&g_app.config);

    if (RegCreateKeyExW(
        HKEY_CURRENT_USER,
        REG_PATH,
        0,
        NULL,
        REG_OPTION_NON_VOLATILE,
        KEY_QUERY_VALUE | KEY_SET_VALUE,
        NULL,
        &registryKey,
        &disposition
    ) != ERROR_SUCCESS) {
        return;
    }

    dataSize = sizeof(DWORD);
    if (RegQueryValueExW(registryKey, REG_KEY_MODIFIERS, NULL, NULL, (LPBYTE)&g_app.config.hotkeyModifiers, &dataSize) != ERROR_SUCCESS) {
        g_app.config.hotkeyModifiers = MOD_CONTROL | MOD_ALT;
        RegSetValueExW(registryKey, REG_KEY_MODIFIERS, 0, REG_DWORD, (const BYTE*)&g_app.config.hotkeyModifiers, sizeof(DWORD));
    }

    dataSize = sizeof(DWORD);
    if (RegQueryValueExW(registryKey, REG_KEY_VIRTUAL_KEY, NULL, NULL, (LPBYTE)&g_app.config.hotkeyVirtualKey, &dataSize) != ERROR_SUCCESS) {
        g_app.config.hotkeyVirtualKey = (DWORD)'V';
        RegSetValueExW(registryKey, REG_KEY_VIRTUAL_KEY, 0, REG_DWORD, (const BYTE*)&g_app.config.hotkeyVirtualKey, sizeof(DWORD));
    }

    RegCloseKey(registryKey);
}

static void AppSaveHotkeyToRegistry(void) {
    HKEY registryKey;

    if (RegOpenKeyExW(HKEY_CURRENT_USER, REG_PATH, 0, KEY_SET_VALUE, &registryKey) != ERROR_SUCCESS) {
        return;
    }

    RegSetValueExW(registryKey, REG_KEY_MODIFIERS, 0, REG_DWORD, (const BYTE*)&g_app.config.hotkeyModifiers, sizeof(DWORD));
    RegSetValueExW(registryKey, REG_KEY_VIRTUAL_KEY, 0, REG_DWORD, (const BYTE*)&g_app.config.hotkeyVirtualKey, sizeof(DWORD));

    RegCloseKey(registryKey);
}

static BOOL AppRegisterWindowClass(HINSTANCE instance) {
    WNDCLASSW windowClass = {0};

    windowClass.lpfnWndProc = AppWindowProc;
    windowClass.hInstance = instance;
    windowClass.lpszClassName = APP_WINDOW_CLASS_NAME;

    if (RegisterClassW(&windowClass)) {
        return TRUE;
    }

    return GetLastError() == ERROR_CLASS_ALREADY_EXISTS;
}

static BOOL AppRegisterHotkey(HWND windowHandle) {
    return RegisterHotKey(
        windowHandle,
        APP_HOTKEY_ID,
        (UINT)g_app.config.hotkeyModifiers,
        (UINT)g_app.config.hotkeyVirtualKey
    );
}

static BOOL AppUnregisterHotkey(HWND windowHandle) {
    return UnregisterHotKey(windowHandle, APP_HOTKEY_ID);
}

static void AppDrainPendingHotkeyMessages(HWND windowHandle) {
    MSG pendingMessage;

    while (PeekMessage(&pendingMessage, windowHandle, WM_HOTKEY, WM_HOTKEY, PM_REMOVE)) {
    }
}

static void AppBuildModifierString(uint32_t modifiers, wchar_t* buffer, size_t capacity) {
    buffer[0] = L'\0';
    if (modifiers & MOD_CONTROL) StringCchCatW(buffer, capacity, L"Ctrl+");
    if (modifiers & MOD_ALT) StringCchCatW(buffer, capacity, L"Alt+");
    if (modifiers & MOD_SHIFT) StringCchCatW(buffer, capacity, L"Shift+");
    if (modifiers & MOD_WIN) StringCchCatW(buffer, capacity, L"Win+");
}

static const wchar_t* AppGetKeyName(uint32_t virtualKey) {
    static wchar_t buffer[16];

    if (virtualKey >= 0x30 && virtualKey <= 0x39) {
        buffer[0] = (wchar_t)virtualKey;
        buffer[1] = L'\0';
        return buffer;
    }
    if (virtualKey >= 0x41 && virtualKey <= 0x5A) {
        buffer[0] = (wchar_t)virtualKey;
        buffer[1] = L'\0';
        return buffer;
    }
    if (virtualKey >= VK_F1 && virtualKey <= VK_F12) {
        StringCchPrintfW(buffer, ARRAYSIZE(buffer), L"F%d", (int)(virtualKey - VK_F1 + 1));
        return buffer;
    }

    StringCchPrintfW(buffer, ARRAYSIZE(buffer), L"0x%02X", virtualKey);
    return buffer;
}

static void AppUpdateTrayTooltip(void) {
    wchar_t modifierText[64];

    AppBuildModifierString(g_app.config.hotkeyModifiers, modifierText, ARRAYSIZE(modifierText));

    StringCchPrintfW(
        g_app.trayIcon.szTip,
        ARRAYSIZE(g_app.trayIcon.szTip),
        L"粘贴助手 (%ls%ls)",
        modifierText,
        AppGetKeyName(g_app.config.hotkeyVirtualKey)
    );
}

static void AppInitializeTrayIcon(HWND windowHandle) {
    ZeroMemory(&g_app.trayIcon, sizeof(g_app.trayIcon));
    g_app.trayIcon.cbSize = sizeof(NOTIFYICONDATA);
    g_app.trayIcon.hWnd = windowHandle;
    g_app.trayIcon.uID = APP_TRAY_ICON_ID;
    g_app.trayIcon.uFlags = NIF_ICON | NIF_MESSAGE | NIF_TIP;
    g_app.trayIcon.uCallbackMessage = WM_APP_TRAYICON;
    g_app.trayIcon.hIcon = LoadIcon(NULL, IDI_APPLICATION);

    AppUpdateTrayTooltip();
    Shell_NotifyIconW(NIM_ADD, &g_app.trayIcon);
}

static void AppRemoveTrayIcon(void) {
    Shell_NotifyIconW(NIM_DELETE, &g_app.trayIcon);
}

static void AppShowTrayMenu(HWND windowHandle) {
    POINT cursorPosition;
    HMENU popupMenu;
    UINT commandId;

    GetCursorPos(&cursorPosition);
    SetForegroundWindow(windowHandle);

    popupMenu = CreatePopupMenu();
    if (!popupMenu) {
        return;
    }

    AppendMenuW(popupMenu, MF_STRING, IDM_ABOUT, L"关于 (About)");
    AppendMenuW(popupMenu, MF_STRING, IDM_HELP, L"使用说明 (Help)");
    AppendMenuW(popupMenu, MF_STRING, IDM_CHECK_UPDATE, L"检查更新 (Latest Release)");
    AppendMenuW(popupMenu, MF_STRING, IDM_DOWNLOAD_PORTABLE, L"下载最新便携版 (Portable EXE)");
    AppendMenuW(popupMenu, MF_STRING, IDM_DOWNLOAD_INSTALLER, L"下载最新安装包 (Installer EXE)");
    AppendMenuW(popupMenu, MF_STRING, IDM_OPEN_REPOSITORY, L"仓库主页 (Repository)");
    AppendMenuW(popupMenu, MF_SEPARATOR, 0, NULL);
    AppendMenuW(popupMenu, MF_STRING, IDM_EXIT, L"退出 (Exit)");

    commandId = TrackPopupMenu(
        popupMenu,
        TPM_RETURNCMD | TPM_NONOTIFY,
        cursorPosition.x,
        cursorPosition.y,
        0,
        windowHandle,
        NULL
    );

    AppHandleTrayCommand(windowHandle, commandId);

    DestroyMenu(popupMenu);
}

static void AppHandleTrayCommand(HWND windowHandle, UINT commandId) {
    switch (commandId) {
        case IDM_ABOUT:
            AppShowAbout();
            break;

        case IDM_HELP:
            AppShowHelp();
            break;

        case IDM_CHECK_UPDATE:
            AppCheckForUpdates();
            break;

        case IDM_DOWNLOAD_PORTABLE:
            AppDownloadLatestPortable();
            break;

        case IDM_DOWNLOAD_INSTALLER:
            AppDownloadLatestInstaller();
            break;

        case IDM_OPEN_REPOSITORY:
            AppOpenRepository();
            break;

        case IDM_CHANGE_HOTKEY: {
            uint32_t oldModifiers = g_app.config.hotkeyModifiers;
            uint32_t oldVirtualKey = g_app.config.hotkeyVirtualKey;

            if (AppShowChangeHotkeyDialog()) {
                AppUnregisterHotkey(windowHandle);

                g_app.config.hotkeyModifiers = g_hotkeyDialogResult.modifiers;
                g_app.config.hotkeyVirtualKey = g_hotkeyDialogResult.virtualKey;

                if (!AppRegisterHotkey(windowHandle)) {
                    g_app.config.hotkeyModifiers = oldModifiers;
                    g_app.config.hotkeyVirtualKey = oldVirtualKey;
                    AppRegisterHotkey(windowHandle);
                    AppShowError(L"\u65b0\u70ed\u952e\u6ce8\u518c\u5931\u8d25\uff0c\u53ef\u80fd\u5df2\u88ab\u5176\u4ed6\u7a0b\u5e8f\u5360\u7528\u3002\u5df2\u6062\u590d\u539f\u8bbe\u7f6e\u3002");
                } else {
                    AppSaveHotkeyToRegistry();
                    AppUpdateTrayTooltip();
                    Shell_NotifyIconW(NIM_MODIFY, &g_app.trayIcon);
                }
            }
            break;
        }

        case IDM_EXIT:
            DestroyWindow(windowHandle);
            break;

        default:
            break;
    }
}

static volatile BOOL g_hotkeyDialogDone;

typedef struct HotkeyDialogResult {
    BOOL confirmed;
    uint32_t modifiers;
    uint32_t virtualKey;
} HotkeyDialogResult;

static HotkeyDialogResult g_hotkeyDialogResult;

static LRESULT CALLBACK AppHotkeyDialogProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    switch (msg) {
        case WM_CREATE: {
            HFONT defaultFont = (HFONT)GetStockObject(DEFAULT_GUI_FONT);
            HWND ctrl;
            HWND comboBox;
            int i, idx, count;
            wchar_t label[8];

            ctrl = CreateWindowExW(0, L"STATIC", L"\u4fee\u9970\u952e:", WS_CHILD | WS_VISIBLE,
                15, 15, 60, 20, hwnd, NULL, g_app.instance, NULL);
            SendMessageW(ctrl, WM_SETFONT, (WPARAM)defaultFont, TRUE);

            ctrl = CreateWindowExW(0, L"BUTTON", L"Ctrl",
                WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_AUTOCHECKBOX,
                80, 15, 55, 20, hwnd, (HMENU)IDC_CHECK_CTRL, g_app.instance, NULL);
            SendMessageW(ctrl, WM_SETFONT, (WPARAM)defaultFont, TRUE);
            if (g_app.config.hotkeyModifiers & MOD_CONTROL)
                SendMessageW(ctrl, BM_SETCHECK, BST_CHECKED, 0);

            ctrl = CreateWindowExW(0, L"BUTTON", L"Alt",
                WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_AUTOCHECKBOX,
                140, 15, 50, 20, hwnd, (HMENU)IDC_CHECK_ALT, g_app.instance, NULL);
            SendMessageW(ctrl, WM_SETFONT, (WPARAM)defaultFont, TRUE);
            if (g_app.config.hotkeyModifiers & MOD_ALT)
                SendMessageW(ctrl, BM_SETCHECK, BST_CHECKED, 0);

            ctrl = CreateWindowExW(0, L"BUTTON", L"Shift",
                WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_AUTOCHECKBOX,
                195, 15, 55, 20, hwnd, (HMENU)IDC_CHECK_SHIFT, g_app.instance, NULL);
            SendMessageW(ctrl, WM_SETFONT, (WPARAM)defaultFont, TRUE);
            if (g_app.config.hotkeyModifiers & MOD_SHIFT)
                SendMessageW(ctrl, BM_SETCHECK, BST_CHECKED, 0);

            ctrl = CreateWindowExW(0, L"BUTTON", L"Win",
                WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_AUTOCHECKBOX,
                255, 15, 50, 20, hwnd, (HMENU)IDC_CHECK_WIN, g_app.instance, NULL);
            SendMessageW(ctrl, WM_SETFONT, (WPARAM)defaultFont, TRUE);
            if (g_app.config.hotkeyModifiers & MOD_WIN)
                SendMessageW(ctrl, BM_SETCHECK, BST_CHECKED, 0);

            ctrl = CreateWindowExW(0, L"STATIC", L"\u89e6\u53d1\u952e:", WS_CHILD | WS_VISIBLE,
                15, 50, 60, 20, hwnd, NULL, g_app.instance, NULL);
            SendMessageW(ctrl, WM_SETFONT, (WPARAM)defaultFont, TRUE);

            comboBox = CreateWindowExW(0, L"COMBOBOX", NULL,
                WS_CHILD | WS_VISIBLE | WS_TABSTOP | CBS_DROPDOWNLIST | WS_VSCROLL,
                80, 47, 120, 200, hwnd, (HMENU)IDC_COMBO_KEY, g_app.instance, NULL);
            SendMessageW(comboBox, WM_SETFONT, (WPARAM)defaultFont, TRUE);

            for (i = 0; i <= 9; i++) {
                StringCchPrintfW(label, ARRAYSIZE(label), L"%d", i);
                idx = (int)SendMessageW(comboBox, CB_ADDSTRING, 0, (LPARAM)label);
                SendMessageW(comboBox, CB_SETITEMDATA, (WPARAM)idx, (LPARAM)(0x30 + i));
            }
            for (i = 0; i < 26; i++) {
                label[0] = L'A' + (wchar_t)i;
                label[1] = L'\0';
                idx = (int)SendMessageW(comboBox, CB_ADDSTRING, 0, (LPARAM)label);
                SendMessageW(comboBox, CB_SETITEMDATA, (WPARAM)idx, (LPARAM)(0x41 + i));
            }
            for (i = 1; i <= 12; i++) {
                StringCchPrintfW(label, ARRAYSIZE(label), L"F%d", i);
                idx = (int)SendMessageW(comboBox, CB_ADDSTRING, 0, (LPARAM)label);
                SendMessageW(comboBox, CB_SETITEMDATA, (WPARAM)idx, (LPARAM)(VK_F1 + i - 1));
            }

            count = (int)SendMessageW(comboBox, CB_GETCOUNT, 0, 0);
            for (i = 0; i < count; i++) {
                if ((uint32_t)SendMessageW(comboBox, CB_GETITEMDATA, (WPARAM)i, 0) == g_app.config.hotkeyVirtualKey) {
                    SendMessageW(comboBox, CB_SETCURSEL, (WPARAM)i, 0);
                    break;
                }
            }

            ctrl = CreateWindowExW(0, L"BUTTON", L"\u786e\u5b9a",
                WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_DEFPUSHBUTTON,
                80, 85, 80, 28, hwnd, (HMENU)IDC_BTN_OK, g_app.instance, NULL);
            SendMessageW(ctrl, WM_SETFONT, (WPARAM)defaultFont, TRUE);

            ctrl = CreateWindowExW(0, L"BUTTON", L"\u53d6\u6d88",
                WS_CHILD | WS_VISIBLE | WS_TABSTOP | BS_PUSHBUTTON,
                175, 85, 80, 28, hwnd, (HMENU)IDC_BTN_CANCEL, g_app.instance, NULL);
            SendMessageW(ctrl, WM_SETFONT, (WPARAM)defaultFont, TRUE);

            return 0;
        }

        case WM_COMMAND: {
            WORD cmdId = LOWORD(wParam);

            if (cmdId == IDC_BTN_OK) {
                HWND comboBox = GetDlgItem(hwnd, IDC_COMBO_KEY);
                int sel = (int)SendMessageW(comboBox, CB_GETCURSEL, 0, 0);
                uint32_t modifiers = 0;

                if (SendMessageW(GetDlgItem(hwnd, IDC_CHECK_CTRL), BM_GETCHECK, 0, 0) == BST_CHECKED)
                    modifiers |= MOD_CONTROL;
                if (SendMessageW(GetDlgItem(hwnd, IDC_CHECK_ALT), BM_GETCHECK, 0, 0) == BST_CHECKED)
                    modifiers |= MOD_ALT;
                if (SendMessageW(GetDlgItem(hwnd, IDC_CHECK_SHIFT), BM_GETCHECK, 0, 0) == BST_CHECKED)
                    modifiers |= MOD_SHIFT;
                if (SendMessageW(GetDlgItem(hwnd, IDC_CHECK_WIN), BM_GETCHECK, 0, 0) == BST_CHECKED)
                    modifiers |= MOD_WIN;

                if (modifiers == 0) {
                    MessageBoxW(hwnd, L"\u8bf7\u81f3\u5c11\u9009\u62e9\u4e00\u4e2a\u4fee\u9970\u952e\uff08Ctrl / Alt / Shift / Win\uff09\u3002",
                        L"\u63d0\u793a", MB_ICONWARNING);
                    return 0;
                }

                if (sel == CB_ERR) {
                    MessageBoxW(hwnd, L"\u8bf7\u9009\u62e9\u4e00\u4e2a\u89e6\u53d1\u952e\u3002", L"\u63d0\u793a", MB_ICONWARNING);
                    return 0;
                }

                g_hotkeyDialogResult.confirmed = TRUE;
                g_hotkeyDialogResult.modifiers = modifiers;
                g_hotkeyDialogResult.virtualKey = (uint32_t)SendMessageW(comboBox, CB_GETITEMDATA, (WPARAM)sel, 0);
                DestroyWindow(hwnd);
                return 0;
            }

            if (cmdId == IDC_BTN_CANCEL) {
                g_hotkeyDialogResult.confirmed = FALSE;
                DestroyWindow(hwnd);
                return 0;
            }
            break;
        }

        case WM_CLOSE:
            g_hotkeyDialogResult.confirmed = FALSE;
            DestroyWindow(hwnd);
            return 0;

        case WM_DESTROY:
            g_hotkeyDialogDone = TRUE;
            return 0;

        default:
            return DefWindowProcW(hwnd, msg, wParam, lParam);
    }
    return DefWindowProcW(hwnd, msg, wParam, lParam);
}

static BOOL AppShowChangeHotkeyDialog(void) {
    WNDCLASSW wc = {0};
    HWND dialogWindow;
    MSG msg;
    RECT screenRect;
    int x, y, width, height;

    wc.lpfnWndProc = AppHotkeyDialogProc;
    wc.hInstance = g_app.instance;
    wc.hbrBackground = (HBRUSH)(COLOR_BTNFACE + 1);
    wc.lpszClassName = HOTKEY_DIALOG_CLASS;
    wc.hCursor = LoadCursor(NULL, IDC_ARROW);
    RegisterClassW(&wc);

    g_hotkeyDialogDone = FALSE;
    g_hotkeyDialogResult.confirmed = FALSE;

    width = 340;
    height = 160;

    SystemParametersInfoW(SPI_GETWORKAREA, 0, &screenRect, 0);
    x = screenRect.left + (screenRect.right - screenRect.left - width) / 2;
    y = screenRect.top + (screenRect.bottom - screenRect.top - height) / 2;

    dialogWindow = CreateWindowExW(
        WS_EX_DLGMODALFRAME | WS_EX_TOPMOST,
        HOTKEY_DIALOG_CLASS,
        L"\u66f4\u6539\u70ed\u952e",
        WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU,
        x, y, width, height,
        NULL,
        NULL,
        g_app.instance,
        NULL
    );

    if (!dialogWindow) return FALSE;

    ShowWindow(dialogWindow, SW_SHOW);
    UpdateWindow(dialogWindow);

    while (!g_hotkeyDialogDone) {
        BOOL ret = GetMessage(&msg, NULL, 0, 0);
        if (ret <= 0) break;
        if (!IsDialogMessage(dialogWindow, &msg)) {
            TranslateMessage(&msg);
            DispatchMessageW(&msg);
        }
    }

    return g_hotkeyDialogResult.confirmed;
}

static wchar_t* AppReadClipboardText(void) {
    HANDLE clipboardData;
    wchar_t* lockedText;
    wchar_t* copiedText;
    size_t textLength;

    if (!OpenClipboard(NULL)) {
        return NULL;
    }

    if (!IsClipboardFormatAvailable(CF_UNICODETEXT)) {
        CloseClipboard();
        return NULL;
    }

    clipboardData = GetClipboardData(CF_UNICODETEXT);
    if (!clipboardData) {
        CloseClipboard();
        return NULL;
    }

    lockedText = (wchar_t*)GlobalLock(clipboardData);
    if (!lockedText) {
        CloseClipboard();
        return NULL;
    }

    textLength = wcslen(lockedText);
    copiedText = (wchar_t*)malloc((textLength + 1) * sizeof(wchar_t));
    if (copiedText) {
        StringCchCopyW(copiedText, textLength + 1, lockedText);
    }

    GlobalUnlock(clipboardData);
    CloseClipboard();
    return copiedText;
}

static void AppSleepMs(void* userData, uint32_t milliseconds) {
    (void)userData;
    Sleep((DWORD)milliseconds);
}

static void AppNotifyPasteStart(void* userData) {
    (void)userData;
    MessageBeep(MB_OK);
}

static void AppNotifyPasteError(void* userData) {
    (void)userData;
    Beep(200, 200);
}

static void AppSendUnicodeCharacter(void* userData, wchar_t character) {
    INPUT inputs[4];
    int count = 0;
    SHORT vkResult;
    BYTE vkCode;
    BYTE shiftState;
    (void)userData;

    ZeroMemory(inputs, sizeof(inputs));

    /* Newline: send as Enter key press (atomic pair) */
    if (character == L'\n') {
        inputs[0].type = INPUT_KEYBOARD;
        inputs[0].ki.wVk = VK_RETURN;
        inputs[1].type = INPUT_KEYBOARD;
        inputs[1].ki.wVk = VK_RETURN;
        inputs[1].ki.dwFlags = KEYEVENTF_KEYUP;
        SendInput(2, inputs, sizeof(INPUT));
        return;
    }

    /* Try mapping character to a real virtual key code via the current keyboard layout.
       This is more compatible with terminals and browsers than KEYEVENTF_UNICODE,
       which generates VK_PACKET that some applications mishandle (e.g. duplicating
       or dropping characters like '-'). Only use this path when no modifier or
       Shift-only is required, to avoid triggering Ctrl/Alt shortcuts. */
    vkResult = VkKeyScanW(character);
    if (vkResult != -1) {
        vkCode = LOBYTE(vkResult);
        shiftState = HIBYTE(vkResult);

        if ((shiftState & ~1) == 0) {
            if (shiftState & 1) {
                inputs[count].type = INPUT_KEYBOARD;
                inputs[count].ki.wVk = VK_SHIFT;
                count++;
            }

            inputs[count].type = INPUT_KEYBOARD;
            inputs[count].ki.wVk = vkCode;
            count++;

            inputs[count].type = INPUT_KEYBOARD;
            inputs[count].ki.wVk = vkCode;
            inputs[count].ki.dwFlags = KEYEVENTF_KEYUP;
            count++;

            if (shiftState & 1) {
                inputs[count].type = INPUT_KEYBOARD;
                inputs[count].ki.wVk = VK_SHIFT;
                inputs[count].ki.dwFlags = KEYEVENTF_KEYUP;
                count++;
            }

            SendInput((UINT)count, inputs, sizeof(INPUT));
            return;
        }
    }

    /* Fallback: Unicode mode for characters that can't be typed with a single key
       (e.g. CJK characters, AltGr-dependent symbols). Still sent as an atomic pair. */
    inputs[0].type = INPUT_KEYBOARD;
    inputs[0].ki.wScan = character;
    inputs[0].ki.dwFlags = KEYEVENTF_UNICODE;
    inputs[1].type = INPUT_KEYBOARD;
    inputs[1].ki.wScan = character;
    inputs[1].ki.dwFlags = KEYEVENTF_UNICODE | KEYEVENTF_KEYUP;
    SendInput(2, inputs, sizeof(INPUT));
}

static DWORD WINAPI AppPasteThreadProc(LPVOID parameter) {
    wchar_t* clipboardSnapshot = (wchar_t*)parameter;
    AppPasteHooks hooks;

    hooks.notifyPasteStart = AppNotifyPasteStart;
    hooks.notifyPasteError = AppNotifyPasteError;
    hooks.sleepMs = AppSleepMs;
    hooks.sendCharacter = AppSendUnicodeCharacter;
    hooks.userData = NULL;

    AppPasteTextSnapshot(clipboardSnapshot, &g_app.config, &hooks);
    free(clipboardSnapshot);
    PostMessage(g_app.messageWindow, WM_APP_PASTE_DONE, 0, 0);

    return 0;
}

static BOOL AppStartPasteOperation(HWND windowHandle) {
    wchar_t* clipboardSnapshot;
    HANDLE threadHandle;

    if (InterlockedCompareExchange(&g_app.isPasting, TRUE, FALSE) != FALSE) {
        return TRUE;
    }

    clipboardSnapshot = AppReadClipboardText();
    if (!clipboardSnapshot) {
        InterlockedExchange(&g_app.isPasting, FALSE);
        AppNotifyPasteError(NULL);
        return FALSE;
    }

    if (!AppUnregisterHotkey(windowHandle)) {
        InterlockedExchange(&g_app.isPasting, FALSE);
        free(clipboardSnapshot);
        AppShowError(L"热键注销失败，无法开始粘贴任务。");
        return FALSE;
    }

    AppDrainPendingHotkeyMessages(windowHandle);

    threadHandle = CreateThread(NULL, 0, AppPasteThreadProc, clipboardSnapshot, 0, NULL);
    if (!threadHandle) {
        InterlockedExchange(&g_app.isPasting, FALSE);
        free(clipboardSnapshot);
        AppRegisterHotkey(windowHandle);
        AppShowError(L"无法创建粘贴线程。");
        return FALSE;
    }

    CloseHandle(threadHandle);
    return TRUE;
}

static void AppFinishPasteOperation(HWND windowHandle) {
    InterlockedExchange(&g_app.isPasting, FALSE);

    if (!AppRegisterHotkey(windowHandle)) {
        AppShowError(L"粘贴完成，但热键重新注册失败，请检查是否被占用。");
    }
}

static LRESULT CALLBACK AppWindowProc(HWND windowHandle, UINT message, WPARAM wParam, LPARAM lParam) {
    switch (message) {
        case WM_CREATE:
            AppInitializeTrayIcon(windowHandle);
            if (!AppRegisterHotkey(windowHandle)) {
                AppShowError(L"热键注册失败，请检查是否被占用。");
                PostQuitMessage(0);
            }
            return 0;

        case WM_HOTKEY:
            if (wParam == APP_HOTKEY_ID) {
                AppStartPasteOperation(windowHandle);
            }
            return 0;

        case WM_APP_PASTE_DONE:
            AppFinishPasteOperation(windowHandle);
            return 0;

        case WM_APP_TRAYICON:
            if (lParam == WM_RBUTTONUP) {
                AppShowTrayMenu(windowHandle);
            }
            return 0;

        case WM_DESTROY:
            AppRemoveTrayIcon();
            AppUnregisterHotkey(windowHandle);
            PostQuitMessage(0);
            return 0;

        default:
            return DefWindowProcW(windowHandle, message, wParam, lParam);
    }
}

int AppRunWin32(HINSTANCE instance) {
    MSG message;

    ZeroMemory(&g_app, sizeof(g_app));
    g_app.instance = instance;
    AppLoadConfigFromRegistry();

    if (!AppRegisterWindowClass(instance)) {
        return 1;
    }

    g_app.messageWindow = CreateWindowW(
        APP_WINDOW_CLASS_NAME,
        APP_WINDOW_TITLE,
        0,
        0,
        0,
        0,
        0,
        HWND_MESSAGE,
        NULL,
        instance,
        NULL
    );

    if (!g_app.messageWindow) {
        return 1;
    }

    while (GetMessage(&message, NULL, 0, 0)) {
        TranslateMessage(&message);
        DispatchMessageW(&message);
    }

    return (int)message.wParam;
}