#ifndef PASTE_APP_H
#define PASTE_APP_H

#include <QObject>
#include <QSystemTrayIcon>
#include <QMenu>
#include <QIcon>
#include <QSettings>
#include <QThread>
#include <QAtomicInt>

#include "platform/InputSimulator.h"

struct AppConfig;

class PasteWorker : public QThread {
    Q_OBJECT
public:
    PasteWorker(const QString &text, const AppConfig *config, InputSimulator *sim, QObject *parent = nullptr);
protected:
    void run() override;
signals:
    void pasteFinished();
private:
    QString m_text;
    const AppConfig *m_config;
    InputSimulator *m_simulator;
};

class PasteApp : public QObject {
    Q_OBJECT
public:
    explicit PasteApp(QObject *parent = nullptr);
    ~PasteApp();

private slots:
    void onTrayActivated(QSystemTrayIcon::ActivationReason reason);
    void onHotkeyTriggered();
    void onPasteFinished();

    void showAbout();
    void showHelp();
    void checkForUpdates();
    void downloadLatestPortable();
    void downloadLatestInstaller();
    void openRepository();
    void changeHotkey();

private:
    void loadConfig();
    void saveHotkeyConfig();
    void setupTrayIcon();
    void setupHotkey();
    void updateTrayTooltip();
    void startPasteOperation();
    QIcon createTrayIcon() const;
    QString buildModifierString(quint32 modifiers) const;
    QString keyName(quint32 vk) const;

    QSystemTrayIcon *m_trayIcon;
    QMenu *m_trayMenu;
    QSettings m_settings;
    AppConfig *m_config;
    InputSimulator *m_simulator;
    QObject *m_hotkey;
    QAtomicInt m_isPasting;
    bool m_hotkeyRegistered;
};

#endif
