#include "platform_win32.h"

#include "app_core.h"
#include "app_metadata.h"

#include <shellapi.h>
#include <stdlib.h>
#include <strsafe.h>

#define WM_APP_TRAYICON (WM_USER + 1)
#define WM_APP_PASTE_DONE (WM_USER + 2)

#define APP_TRAY_ICON_ID 1
#define APP_HOTKEY_ID 100

#define IDM_CHECK_UPDATE 2000
#define IDM_HELP 2001
#define IDM_EXIT 2002

#define APP_WINDOW_CLASS_NAME L"AutoPasteToolClass"
#define APP_WINDOW_TITLE L"PasteTool"

#define REG_PATH L"Software\\AutoPasteTool"
#define REG_KEY_MODIFIERS L"KeyModifier"
#define REG_KEY_VIRTUAL_KEY L"KeyVirtual"

typedef struct Win32AppState {
    HINSTANCE instance;
    HWND messageWindow;
    LONG isPasting;
    NOTIFYICONDATA trayIcon;
    AppConfig config;
} Win32AppState;

static Win32AppState g_app;

static LRESULT CALLBACK AppWindowProc(HWND windowHandle, UINT message, WPARAM wParam, LPARAM lParam);

static void AppShowError(const wchar_t* message);
static void AppShowHelp(void);
static void AppCheckForUpdates(void);
static void AppLoadConfigFromRegistry(void);
static BOOL AppRegisterWindowClass(HINSTANCE instance);
static BOOL AppRegisterHotkey(HWND windowHandle);
static BOOL AppUnregisterHotkey(HWND windowHandle);
static void AppDrainPendingHotkeyMessages(HWND windowHandle);
static void AppInitializeTrayIcon(HWND windowHandle);
static void AppUpdateTrayTooltip(void);
static void AppRemoveTrayIcon(void);
static void AppShowTrayMenu(HWND windowHandle);
static wchar_t* AppReadClipboardText(void);
static void AppOpenUrl(const wchar_t* url);
static void AppSleepMs(void* userData, uint32_t milliseconds);
static void AppNotifyPasteStart(void* userData);
static void AppNotifyPasteError(void* userData);
static void AppSendUnicodeCharacter(void* userData, wchar_t character);
static DWORD WINAPI AppPasteThreadProc(LPVOID parameter);
static BOOL AppStartPasteOperation(HWND windowHandle);
static void AppFinishPasteOperation(HWND windowHandle);

static void AppShowError(const wchar_t* message) {
    MessageBoxW(NULL, message, L"错误", MB_ICONERROR);
}

static void AppShowHelp(void) {
    wchar_t message[512];

    StringCchPrintfW(
        message,
        ARRAYSIZE(message),
        L"【%ls %ls 使用说明】\n\n"
        L"1. 复制：先复制你要输入的代码或文本。\n"
        L"2. 触发：按下快捷键 Ctrl + Alt + %lc。\n"
        L"3. 准备：听到提示音后，你有 %lu 秒切到目标输入框。\n"
        L"4. 粘贴：程序会自动模拟键盘输入。\n\n"
        L"注意事项：\n"
        L"- 输入期间会暂时禁用热键，避免重复触发。\n"
        L"- 程序在系统托盘运行，右键图标可查看帮助或退出。\n"
        L"- 检查更新会打开 GitHub 最新 release 页面。\n"
        L"- 新版本建议通过 GitHub release 附件下载安装。\n"
        L"- 核心粘贴逻辑已拆到 app_core.c，便于后续做跨平台版本。",
        APP_NAME,
        APP_VERSION,
        (wchar_t)g_app.config.hotkeyVirtualKey,
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

static void AppCheckForUpdates(void) {
    AppOpenUrl(APP_LATEST_RELEASE_URL);
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

static void AppUpdateTrayTooltip(void) {
    StringCchPrintfW(
        g_app.trayIcon.szTip,
        ARRAYSIZE(g_app.trayIcon.szTip),
        L"粘贴助手 (Ctrl+Alt+%lc)",
        (wchar_t)g_app.config.hotkeyVirtualKey
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
    Shell_NotifyIcon(NIM_ADD, &g_app.trayIcon);
}

static void AppRemoveTrayIcon(void) {
    Shell_NotifyIcon(NIM_DELETE, &g_app.trayIcon);
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

    AppendMenuW(popupMenu, MF_STRING, IDM_HELP, L"使用说明 (Help)");
    AppendMenuW(popupMenu, MF_STRING, IDM_CHECK_UPDATE, L"检查更新 (Latest Release)");
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

    if (commandId == IDM_HELP) {
        AppShowHelp();
    } else if (commandId == IDM_CHECK_UPDATE) {
        AppCheckForUpdates();
    } else if (commandId == IDM_EXIT) {
        DestroyWindow(windowHandle);
    }

    DestroyMenu(popupMenu);
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
    INPUT input = {0};
    (void)userData;

    input.type = INPUT_KEYBOARD;
    if (character == L'\n') {
        input.ki.wVk = VK_RETURN;
        SendInput(1, &input, sizeof(INPUT));
        input.ki.dwFlags = KEYEVENTF_KEYUP;
        SendInput(1, &input, sizeof(INPUT));
        return;
    }

    input.ki.wScan = character;
    input.ki.dwFlags = KEYEVENTF_UNICODE;
    SendInput(1, &input, sizeof(INPUT));

    input.ki.dwFlags = KEYEVENTF_UNICODE | KEYEVENTF_KEYUP;
    SendInput(1, &input, sizeof(INPUT));
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