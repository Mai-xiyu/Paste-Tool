#include "PasteApp.h"
#include "HotkeyDialog.h"
#include "UpdateChecker.h"

#include <QApplication>
#include <QClipboard>
#include <QDesktopServices>
#include <QMessageBox>
#include <QUrl>
#include <QStandardPaths>
#include <QDir>

#include <QHotkey>

extern "C" {
#include "app_core.h"
#include "app_metadata.h"
}

/* C callback trampolines for AppPasteTextSnapshot */
static InputSimulator *g_currentSimulator = nullptr;

static void hookSendCharacter(void *, wchar_t ch) {
    if (g_currentSimulator) g_currentSimulator->sendCharacter(ch);
}
static void hookSleepMs(void *, uint32_t ms) {
    QThread::msleep(ms);
}
static void hookNotifyPasteStart(void *) {
    if (g_currentSimulator) g_currentSimulator->notifyPasteStart();
}
static void hookNotifyPasteError(void *) {
    if (g_currentSimulator) g_currentSimulator->notifyPasteError();
}

/* ----- PasteWorker ----- */

PasteWorker::PasteWorker(const QString &text, const AppConfig *config, InputSimulator *sim, QObject *parent)
    : QThread(parent), m_text(text), m_config(config), m_simulator(sim) {}

void PasteWorker::run() {
    g_currentSimulator = m_simulator;

    AppPasteHooks hooks{};
    hooks.sendCharacter = hookSendCharacter;
    hooks.sleepMs = hookSleepMs;
    hooks.notifyPasteStart = hookNotifyPasteStart;
    hooks.notifyPasteError = hookNotifyPasteError;
    hooks.userData = nullptr;

    auto wstr = m_text.toStdWString();
    AppPasteTextSnapshot(wstr.c_str(), m_config, &hooks);

    emit pasteFinished();
}

/* ----- PasteApp ----- */

PasteApp::PasteApp(QObject *parent)
    : QObject(parent)
    , m_trayIcon(nullptr)
    , m_trayMenu(nullptr)
    , m_settings(QStringLiteral("Mai-xiyu"), QStringLiteral("PasteTool"))
    , m_config(new AppConfig)
    , m_simulator(InputSimulator::create())
    , m_hotkey(nullptr)
    , m_isPasting(0)
    , m_hotkeyRegistered(false)
{
    AppConfigInitDefaults(m_config);
    loadConfig();
    setupTrayIcon();
    setupHotkey();
}

PasteApp::~PasteApp() {
    delete m_simulator;
    delete m_config;
}

void PasteApp::loadConfig() {
    if (m_settings.contains(QStringLiteral("hotkey/modifiers"))) {
        m_config->hotkeyModifiers = m_settings.value(QStringLiteral("hotkey/modifiers")).toUInt();
    }
    if (m_settings.contains(QStringLiteral("hotkey/virtualKey"))) {
        m_config->hotkeyVirtualKey = m_settings.value(QStringLiteral("hotkey/virtualKey")).toUInt();
    }
}

void PasteApp::saveHotkeyConfig() {
    m_settings.setValue(QStringLiteral("hotkey/modifiers"), m_config->hotkeyModifiers);
    m_settings.setValue(QStringLiteral("hotkey/virtualKey"), m_config->hotkeyVirtualKey);
}

static Qt::KeyboardModifiers modifiersFromConfig(quint32 mods) {
    Qt::KeyboardModifiers qtMods;
    if (mods & 0x0002) qtMods |= Qt::ControlModifier;  // MOD_CONTROL
    if (mods & 0x0001) qtMods |= Qt::AltModifier;       // MOD_ALT
    if (mods & 0x0004) qtMods |= Qt::ShiftModifier;     // MOD_SHIFT
    if (mods & 0x0008) qtMods |= Qt::MetaModifier;      // MOD_WIN
    return qtMods;
}

static Qt::Key keyFromVk(quint32 vk) {
    if (vk >= 0x30 && vk <= 0x39) return static_cast<Qt::Key>(Qt::Key_0 + (vk - 0x30));
    if (vk >= 0x41 && vk <= 0x5A) return static_cast<Qt::Key>(Qt::Key_A + (vk - 0x41));
    if (vk >= 0x70 && vk <= 0x7B) return static_cast<Qt::Key>(Qt::Key_F1 + (vk - 0x70));
    return static_cast<Qt::Key>(vk);
}

void PasteApp::setupHotkey() {
    delete m_hotkey;
    m_hotkey = nullptr;

    auto *hotkey = new QHotkey(
        QKeySequence(static_cast<int>(modifiersFromConfig(m_config->hotkeyModifiers)) |
                     static_cast<int>(keyFromVk(m_config->hotkeyVirtualKey))),
        true, this
    );

    if (hotkey->isRegistered()) {
        m_hotkeyRegistered = true;
        m_hotkey = hotkey;
        connect(hotkey, &QHotkey::activated, this, &PasteApp::onHotkeyTriggered);
    } else {
        delete hotkey;
        m_hotkeyRegistered = false;

        QMessageBox::warning(nullptr,
            QStringLiteral("热键冲突"),
            QStringLiteral("默认热键已被其他程序占用，请设置新的快捷键。"));

        HotkeyDialog dlg(m_config->hotkeyModifiers, m_config->hotkeyVirtualKey);
        if (dlg.exec() == QDialog::Accepted) {
            m_config->hotkeyModifiers = dlg.selectedModifiers();
            m_config->hotkeyVirtualKey = dlg.selectedVirtualKey();

            auto *newHotkey = new QHotkey(
                QKeySequence(static_cast<int>(modifiersFromConfig(m_config->hotkeyModifiers)) |
                             static_cast<int>(keyFromVk(m_config->hotkeyVirtualKey))),
                true, this
            );

            if (newHotkey->isRegistered()) {
                m_hotkeyRegistered = true;
                m_hotkey = newHotkey;
                connect(newHotkey, &QHotkey::activated, this, &PasteApp::onHotkeyTriggered);
                saveHotkeyConfig();
            } else {
                delete newHotkey;
                QMessageBox::critical(nullptr,
                    QStringLiteral("错误"),
                    QStringLiteral("新热键也注册失败，程序将以无热键模式运行。\n请通过托盘菜单「更改热键」重新设置。"));
            }
        } else {
            QMessageBox::information(nullptr,
                QStringLiteral("提示"),
                QStringLiteral("程序将以无热键模式运行。\n可随时通过托盘菜单「更改热键」设置快捷键。"));
        }
    }

    updateTrayTooltip();
}

void PasteApp::setupTrayIcon() {
    m_trayMenu = new QMenu();

    m_trayMenu->addAction(QStringLiteral("关于 (About)"), this, &PasteApp::showAbout);
    m_trayMenu->addAction(QStringLiteral("使用说明 (Help)"), this, &PasteApp::showHelp);
    m_trayMenu->addAction(QStringLiteral("更改热键 (Change Hotkey)"), this, &PasteApp::changeHotkey);
    m_trayMenu->addSeparator();
    m_trayMenu->addAction(QStringLiteral("检查更新 (Check Update)"), this, &PasteApp::checkForUpdates);
    m_trayMenu->addAction(QStringLiteral("下载最新便携版 (Portable EXE)"), this, &PasteApp::downloadLatestPortable);
    m_trayMenu->addAction(QStringLiteral("下载最新安装包 (Installer EXE)"), this, &PasteApp::downloadLatestInstaller);
    m_trayMenu->addAction(QStringLiteral("仓库主页 (Repository)"), this, &PasteApp::openRepository);
    m_trayMenu->addSeparator();
    m_trayMenu->addAction(QStringLiteral("退出 (Exit)"), qApp, &QApplication::quit);

    m_trayIcon = new QSystemTrayIcon(QIcon::fromTheme(QStringLiteral("application-x-executable")), this);
    m_trayIcon->setContextMenu(m_trayMenu);
    connect(m_trayIcon, &QSystemTrayIcon::activated, this, &PasteApp::onTrayActivated);
    m_trayIcon->show();
}

void PasteApp::updateTrayTooltip() {
    if (!m_hotkeyRegistered) {
        m_trayIcon->setToolTip(QStringLiteral("粘贴助手 (未设置热键)"));
        return;
    }

    m_trayIcon->setToolTip(QStringLiteral("粘贴助手 (%1%2)")
        .arg(buildModifierString(m_config->hotkeyModifiers),
             keyName(m_config->hotkeyVirtualKey)));
}

QString PasteApp::buildModifierString(quint32 modifiers) const {
    QString s;
    if (modifiers & 0x0002) s += QStringLiteral("Ctrl+");
    if (modifiers & 0x0001) s += QStringLiteral("Alt+");
    if (modifiers & 0x0004) s += QStringLiteral("Shift+");
    if (modifiers & 0x0008) s += QStringLiteral("Win+");
    return s;
}

QString PasteApp::keyName(quint32 vk) const {
    if (vk >= 0x30 && vk <= 0x39) return QString(QChar(vk));
    if (vk >= 0x41 && vk <= 0x5A) return QString(QChar(vk));
    if (vk >= 0x70 && vk <= 0x7B) return QStringLiteral("F%1").arg(vk - 0x70 + 1);
    return QStringLiteral("0x%1").arg(vk, 2, 16, QLatin1Char('0'));
}

void PasteApp::onTrayActivated(QSystemTrayIcon::ActivationReason reason) {
    Q_UNUSED(reason);
}

void PasteApp::onHotkeyTriggered() {
    startPasteOperation();
}

void PasteApp::startPasteOperation() {
    if (!m_isPasting.testAndSetRelaxed(0, 1))
        return;

    QString clipText = QApplication::clipboard()->text();
    if (clipText.isEmpty()) {
        m_isPasting.storeRelaxed(0);
        m_simulator->notifyPasteError();
        return;
    }

    /* Temporarily unregister hotkey to prevent re-triggering during paste */
    if (m_hotkey) {
        auto *hk = qobject_cast<QHotkey*>(m_hotkey);
        if (hk) hk->setRegistered(false);
    }

    auto *worker = new PasteWorker(clipText, m_config, m_simulator, this);
    connect(worker, &PasteWorker::pasteFinished, this, &PasteApp::onPasteFinished);
    connect(worker, &PasteWorker::finished, worker, &QObject::deleteLater);
    worker->start();
}

void PasteApp::onPasteFinished() {
    m_isPasting.storeRelaxed(0);

    if (m_hotkey) {
        auto *hk = qobject_cast<QHotkey*>(m_hotkey);
        if (hk) {
            hk->setRegistered(true);
            m_hotkeyRegistered = hk->isRegistered();
            if (!m_hotkeyRegistered) {
                QMessageBox::warning(nullptr,
                    QStringLiteral("错误"),
                    QStringLiteral("粘贴完成，但热键重新注册失败，请检查是否被占用。"));
            }
        }
    }
}

void PasteApp::showAbout() {
    QString msg = QStringLiteral(
        "%1\n版本：%2\n\n"
        "仓库主页：\n%3\n\n"
        "更新检查：\n%4\n\n"
        "新版本建议从 GitHub Release 页面下载安装。")
        .arg(QString::fromWCharArray(APP_NAME),
             QString::fromWCharArray(APP_VERSION),
             QString::fromWCharArray(APP_REPOSITORY_URL),
             QString::fromWCharArray(APP_LATEST_RELEASE_URL));

    QMessageBox::about(nullptr, QStringLiteral("关于"), msg);
}

void PasteApp::showHelp() {
    QString mods = buildModifierString(m_config->hotkeyModifiers);
    QString key = keyName(m_config->hotkeyVirtualKey);

    QString msg = QStringLiteral(
        "【%1 %2 使用说明】\n\n"
        "1. 复制：先复制你要输入的代码或文本。\n"
        "2. 触发：按下快捷键 %3%4。\n"
        "3. 准备：听到提示音后，你有 %5 秒切到目标输入框。\n"
        "4. 粘贴：程序会自动模拟键盘输入。\n\n"
        "注意事项：\n"
        "- 输入期间会暂时禁用热键，避免重复触发。\n"
        "- 程序在系统托盘运行，右键图标可查看帮助或退出。\n"
        "- 可通过右键菜单「更改热键」自定义快捷键组合。\n"
        "- 检查更新会查询 GitHub 最新版本并提示下载。")
        .arg(QString::fromWCharArray(APP_NAME),
             QString::fromWCharArray(APP_VERSION),
             mods, key,
             QString::number(m_config->startDelayMs / 1000));

    QMessageBox::information(nullptr, QStringLiteral("帮助"), msg);
}

void PasteApp::checkForUpdates() {
    UpdateChecker::checkForUpdates(
        QString::fromWCharArray(APP_VERSION),
        QString::fromWCharArray(APP_LATEST_RELEASE_URL));
}

void PasteApp::downloadLatestPortable() {
    UpdateChecker::downloadAsset(
        QString::fromWCharArray(APP_LATEST_PORTABLE_DOWNLOAD_URL),
        QStringLiteral("paste_tool-latest-windows-x64.exe"),
        false);
}

void PasteApp::downloadLatestInstaller() {
    UpdateChecker::downloadAsset(
        QString::fromWCharArray(APP_LATEST_INSTALLER_DOWNLOAD_URL),
        QStringLiteral("paste_tool-installer-latest.exe"),
        true);
}

void PasteApp::openRepository() {
    QDesktopServices::openUrl(QUrl(QString::fromWCharArray(APP_REPOSITORY_URL)));
}

void PasteApp::changeHotkey() {
    quint32 oldMods = m_config->hotkeyModifiers;
    quint32 oldKey = m_config->hotkeyVirtualKey;

    HotkeyDialog dlg(m_config->hotkeyModifiers, m_config->hotkeyVirtualKey);
    if (dlg.exec() != QDialog::Accepted)
        return;

    /* Unregister old hotkey */
    if (m_hotkey) {
        auto *hk = qobject_cast<QHotkey*>(m_hotkey);
        if (hk) hk->setRegistered(false);
        delete m_hotkey;
        m_hotkey = nullptr;
    }

    m_config->hotkeyModifiers = dlg.selectedModifiers();
    m_config->hotkeyVirtualKey = dlg.selectedVirtualKey();

    auto *newHotkey = new QHotkey(
        QKeySequence(static_cast<int>(modifiersFromConfig(m_config->hotkeyModifiers)) |
                     static_cast<int>(keyFromVk(m_config->hotkeyVirtualKey))),
        true, this
    );

    if (newHotkey->isRegistered()) {
        m_hotkeyRegistered = true;
        m_hotkey = newHotkey;
        connect(newHotkey, &QHotkey::activated, this, &PasteApp::onHotkeyTriggered);
        saveHotkeyConfig();
        updateTrayTooltip();
    } else {
        delete newHotkey;
        /* Revert */
        m_config->hotkeyModifiers = oldMods;
        m_config->hotkeyVirtualKey = oldKey;

        auto *revert = new QHotkey(
            QKeySequence(static_cast<int>(modifiersFromConfig(oldMods)) |
                         static_cast<int>(keyFromVk(oldKey))),
            true, this
        );

        if (revert->isRegistered()) {
            m_hotkeyRegistered = true;
            m_hotkey = revert;
            connect(revert, &QHotkey::activated, this, &PasteApp::onHotkeyTriggered);
        } else {
            delete revert;
            m_hotkeyRegistered = false;
        }

        QMessageBox::critical(nullptr,
            QStringLiteral("错误"),
            QStringLiteral("新热键注册失败，可能已被其他程序占用。已恢复原设置。"));
        updateTrayTooltip();
    }
}
