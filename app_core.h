#ifndef APP_CORE_H
#define APP_CORE_H

#include <stddef.h>
#include <stdint.h>
#include <wchar.h>

typedef struct AppConfig {
    uint32_t hotkeyModifiers;
    uint32_t hotkeyVirtualKey;
    uint32_t startDelayMs;
    uint32_t interKeyDelayMs;
    size_t batchSize;
    uint32_t batchPauseMs;
} AppConfig;

typedef struct AppPasteHooks {
    void (*notifyPasteStart)(void* userData);
    void (*notifyPasteError)(void* userData);
    void (*sleepMs)(void* userData, uint32_t milliseconds);
    void (*sendCharacter)(void* userData, wchar_t character);
    void* userData;
} AppPasteHooks;

void AppConfigInitDefaults(AppConfig* config);
void AppPasteTextSnapshot(const wchar_t* text, const AppConfig* config, const AppPasteHooks* hooks);

#endif