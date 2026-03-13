#include "platform/InputSimulator.h"

#include <QtGlobal>

#ifdef Q_OS_WIN
#include <windows.h>
#else
#include <QApplication>
#endif

#ifdef Q_OS_WIN

class InputSimulatorWin : public InputSimulator {
public:
    void sendCharacter(wchar_t character) override;
    void notifyPasteStart() override;
    void notifyPasteError() override;
};

void InputSimulatorWin::sendCharacter(wchar_t character) {
    INPUT inputs[4];
    int count = 0;

    ZeroMemory(inputs, sizeof(inputs));

    if (character == L'\n') {
        inputs[0].type = INPUT_KEYBOARD;
        inputs[0].ki.wVk = VK_RETURN;
        inputs[1].type = INPUT_KEYBOARD;
        inputs[1].ki.wVk = VK_RETURN;
        inputs[1].ki.dwFlags = KEYEVENTF_KEYUP;
        SendInput(2, inputs, sizeof(INPUT));
        return;
    }

    SHORT vkResult = VkKeyScanW(character);
    if (vkResult != -1) {
        BYTE vkCode = LOBYTE(vkResult);
        BYTE shiftState = HIBYTE(vkResult);

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

            SendInput(static_cast<UINT>(count), inputs, sizeof(INPUT));
            return;
        }
    }

    inputs[0].type = INPUT_KEYBOARD;
    inputs[0].ki.wScan = static_cast<WORD>(character);
    inputs[0].ki.dwFlags = KEYEVENTF_UNICODE;
    inputs[1].type = INPUT_KEYBOARD;
    inputs[1].ki.wScan = static_cast<WORD>(character);
    inputs[1].ki.dwFlags = KEYEVENTF_UNICODE | KEYEVENTF_KEYUP;
    SendInput(2, inputs, sizeof(INPUT));
}

void InputSimulatorWin::notifyPasteStart() {
    MessageBeep(MB_OK);
}

void InputSimulatorWin::notifyPasteError() {
    Beep(200, 200);
}

InputSimulator* InputSimulator::create() {
    return new InputSimulatorWin();
}

#endif
