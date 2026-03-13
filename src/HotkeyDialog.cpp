#include "HotkeyDialog.h"

#include <QGridLayout>
#include <QLabel>
#include <QDialogButtonBox>
#include <QMessageBox>
#include <QPushButton>

HotkeyDialog::HotkeyDialog(quint32 currentModifiers, quint32 currentVirtualKey, QWidget *parent)
    : QDialog(parent)
{
    setWindowTitle(QStringLiteral("更改热键"));
    setFixedSize(340, 160);

    auto *layout = new QGridLayout(this);

    layout->addWidget(new QLabel(QStringLiteral("修饰键:")), 0, 0);

    m_checkCtrl = new QCheckBox(QStringLiteral("Ctrl"), this);
    m_checkAlt = new QCheckBox(QStringLiteral("Alt"), this);
    m_checkShift = new QCheckBox(QStringLiteral("Shift"), this);
    m_checkWin = new QCheckBox(QStringLiteral("Win"), this);

    m_checkCtrl->setChecked(currentModifiers & 0x0002);  // MOD_CONTROL
    m_checkAlt->setChecked(currentModifiers & 0x0001);   // MOD_ALT
    m_checkShift->setChecked(currentModifiers & 0x0004); // MOD_SHIFT
    m_checkWin->setChecked(currentModifiers & 0x0008);   // MOD_WIN

    layout->addWidget(m_checkCtrl, 0, 1);
    layout->addWidget(m_checkAlt, 0, 2);
    layout->addWidget(m_checkShift, 0, 3);
    layout->addWidget(m_checkWin, 0, 4);

    layout->addWidget(new QLabel(QStringLiteral("触发键:")), 1, 0);

    m_comboKey = new QComboBox(this);

    for (int i = 0; i <= 9; i++) {
        m_comboKey->addItem(QString::number(i), 0x30 + i);
    }
    for (int i = 0; i < 26; i++) {
        m_comboKey->addItem(QString(QChar('A' + i)), 0x41 + i);
    }
    for (int i = 1; i <= 12; i++) {
        m_comboKey->addItem(QStringLiteral("F%1").arg(i), 0x70 + i - 1);  // VK_F1 = 0x70
    }

    /* Select current key */
    for (int i = 0; i < m_comboKey->count(); i++) {
        if (m_comboKey->itemData(i).toUInt() == currentVirtualKey) {
            m_comboKey->setCurrentIndex(i);
            break;
        }
    }

    layout->addWidget(m_comboKey, 1, 1, 1, 4);

    auto *buttons = new QDialogButtonBox(QDialogButtonBox::Ok | QDialogButtonBox::Cancel, this);
    buttons->button(QDialogButtonBox::Ok)->setText(QStringLiteral("确定"));
    buttons->button(QDialogButtonBox::Cancel)->setText(QStringLiteral("取消"));
    layout->addWidget(buttons, 2, 0, 1, 5);

    connect(buttons, &QDialogButtonBox::accepted, this, [this]() {
        if (selectedModifiers() == 0) {
            QMessageBox::warning(this,
                QStringLiteral("提示"),
                QStringLiteral("请至少选择一个修饰键（Ctrl / Alt / Shift / Win）。"));
            return;
        }
        accept();
    });
    connect(buttons, &QDialogButtonBox::rejected, this, &QDialog::reject);
}

quint32 HotkeyDialog::selectedModifiers() const {
    quint32 mods = 0;
    if (m_checkCtrl->isChecked())  mods |= 0x0002;
    if (m_checkAlt->isChecked())   mods |= 0x0001;
    if (m_checkShift->isChecked()) mods |= 0x0004;
    if (m_checkWin->isChecked())   mods |= 0x0008;
    return mods;
}

quint32 HotkeyDialog::selectedVirtualKey() const {
    return m_comboKey->currentData().toUInt();
}
