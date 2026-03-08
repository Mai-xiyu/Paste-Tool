/* 编译命令 (GCC/MinGW):
   gcc paste_tool.c platform_win32.c app_core.c -o paste_tool.exe -mwindows -lshell32

   注意：必须添加 -lshell32 以支持托盘图标功能
*/

#define UNICODE
#define _UNICODE

#include "platform_win32.h"

int WINAPI WinMain(HINSTANCE instance, HINSTANCE previousInstance, LPSTR commandLine, int showCommand) {
    UNREFERENCED_PARAMETER(previousInstance);
    UNREFERENCED_PARAMETER(commandLine);
    UNREFERENCED_PARAMETER(showCommand);

    return AppRunWin32(instance);
}
