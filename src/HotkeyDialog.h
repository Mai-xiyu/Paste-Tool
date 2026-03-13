#ifndef HOTKEY_DIALOG_H
#define HOTKEY_DIALOG_H

#include <QDialog>
#include <QCheckBox>
#include <QComboBox>
#include <cstdint>

class HotkeyDialog : public QDialog {
    Q_OBJECT
public:
    explicit HotkeyDialog(quint32 currentModifiers, quint32 currentVirtualKey,
                          QWidget *parent = nullptr);

    quint32 selectedModifiers() const;
    quint32 selectedVirtualKey() const;

private:
    QCheckBox *m_checkCtrl;
    QCheckBox *m_checkAlt;
    QCheckBox *m_checkShift;
    QCheckBox *m_checkWin;
    QComboBox *m_comboKey;
};

#endif
