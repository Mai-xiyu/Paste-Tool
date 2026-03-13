#ifndef INPUT_SIMULATOR_H
#define INPUT_SIMULATOR_H

#include <cstdint>

class InputSimulator {
public:
    virtual ~InputSimulator() = default;
    virtual void sendCharacter(wchar_t character) = 0;
    virtual void notifyPasteStart() = 0;
    virtual void notifyPasteError() = 0;

    static InputSimulator* create();
};

#endif
