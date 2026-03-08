#include "app_core.h"

#define DEFAULT_HOTKEY_MODIFIERS 0x0003U
#define DEFAULT_HOTKEY_VIRTUAL_KEY ((uint32_t)'V')
#define DEFAULT_START_DELAY_MS 3000U
#define DEFAULT_INTER_KEY_DELAY_MS 8U
#define DEFAULT_BATCH_SIZE 50U
#define DEFAULT_BATCH_PAUSE_MS 20U

void AppConfigInitDefaults(AppConfig* config) {
    if (!config) {
        return;
    }

    config->hotkeyModifiers = DEFAULT_HOTKEY_MODIFIERS;
    config->hotkeyVirtualKey = DEFAULT_HOTKEY_VIRTUAL_KEY;
    config->startDelayMs = DEFAULT_START_DELAY_MS;
    config->interKeyDelayMs = DEFAULT_INTER_KEY_DELAY_MS;
    config->batchSize = DEFAULT_BATCH_SIZE;
    config->batchPauseMs = DEFAULT_BATCH_PAUSE_MS;
}

void AppPasteTextSnapshot(const wchar_t* text, const AppConfig* config, const AppPasteHooks* hooks) {
    size_t index;

    if (!text || !config || !hooks || !hooks->sleepMs || !hooks->sendCharacter) {
        if (hooks && hooks->notifyPasteError) {
            hooks->notifyPasteError(hooks->userData);
        }
        return;
    }

    if (hooks->notifyPasteStart) {
        hooks->notifyPasteStart(hooks->userData);
    }

    hooks->sleepMs(hooks->userData, config->startDelayMs);

    for (index = 0; text[index] != L'\0'; ++index) {
        if (text[index] == L'\r') {
            continue;
        }

        hooks->sendCharacter(hooks->userData, text[index]);
        hooks->sleepMs(hooks->userData, config->interKeyDelayMs);

        if (index > 0 && config->batchSize > 0 && index % config->batchSize == 0) {
            hooks->sleepMs(hooks->userData, config->batchPauseMs);
        }
    }
}