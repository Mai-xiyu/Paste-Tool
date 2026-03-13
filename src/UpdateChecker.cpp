#include "UpdateChecker.h"

#include <QNetworkAccessManager>
#include <QNetworkRequest>
#include <QNetworkReply>
#include <QJsonDocument>
#include <QJsonObject>
#include <QEventLoop>
#include <QMessageBox>
#include <QDesktopServices>
#include <QStandardPaths>
#include <QDir>
#include <QFile>
#include <QUrl>

int UpdateChecker::compareVersions(const QString &current, const QString &latest) {
    auto parse = [](const QString &v) -> QList<int> {
        QString s = v;
        if (s.startsWith('v', Qt::CaseInsensitive)) s = s.mid(1);
        QList<int> parts;
        for (auto &p : s.split('.'))
            parts.append(p.toInt());
        while (parts.size() < 3) parts.append(0);
        return parts;
    };

    auto c = parse(current);
    auto l = parse(latest);

    for (int i = 0; i < 3; i++) {
        if (l[i] != c[i]) return l[i] - c[i];
    }
    return 0;
}

void UpdateChecker::checkForUpdates(const QString &currentVersion, const QString &latestReleaseUrl) {
    QNetworkAccessManager mgr;
    QNetworkRequest req(QUrl(QStringLiteral("https://api.github.com/repos/Mai-xiyu/Paste-Tool/releases/latest")));
    req.setRawHeader("User-Agent", "PasteTool");
    req.setTransferTimeout(10000);

    QEventLoop loop;
    auto *reply = mgr.get(req);
    QObject::connect(reply, &QNetworkReply::finished, &loop, &QEventLoop::quit);
    loop.exec();

    if (reply->error() != QNetworkReply::NoError) {
        reply->deleteLater();
        int answer = QMessageBox::warning(nullptr,
            QStringLiteral("检查更新"),
            QStringLiteral("无法获取最新版本信息，请检查网络连接。\n\n是否手动打开 GitHub Release 页面？"),
            QMessageBox::Yes | QMessageBox::No);
        if (answer == QMessageBox::Yes) {
            QDesktopServices::openUrl(QUrl(latestReleaseUrl));
        }
        return;
    }

    auto doc = QJsonDocument::fromJson(reply->readAll());
    reply->deleteLater();

    QString tagName = doc.object().value(QStringLiteral("tag_name")).toString();
    if (tagName.isEmpty()) {
        QMessageBox::warning(nullptr,
            QStringLiteral("检查更新"),
            QStringLiteral("无法解析版本信息。"));
        return;
    }

    if (compareVersions(currentVersion, tagName) > 0) {
        int answer = QMessageBox::information(nullptr,
            QStringLiteral("检查更新"),
            QStringLiteral("发现新版本 %1！\n当前版本：%2\n\n是否打开下载页面？").arg(tagName, currentVersion),
            QMessageBox::Yes | QMessageBox::No);
        if (answer == QMessageBox::Yes) {
            QDesktopServices::openUrl(QUrl(latestReleaseUrl));
        }
    } else {
        QMessageBox::information(nullptr,
            QStringLiteral("检查更新"),
            QStringLiteral("当前版本 %1 已是最新版本。").arg(currentVersion));
    }
}

void UpdateChecker::downloadAsset(const QString &downloadUrl, const QString &fileName, bool launchAfterDownload) {
    QString downloadDir = QStandardPaths::writableLocation(QStandardPaths::DownloadLocation);
    if (downloadDir.isEmpty()) {
        QMessageBox::critical(nullptr,
            QStringLiteral("错误"),
            QStringLiteral("无法定位下载目录。"));
        return;
    }

    QDir().mkpath(downloadDir);
    QString outputPath = downloadDir + QDir::separator() + fileName;

    QNetworkAccessManager mgr;
    QNetworkRequest req{QUrl(downloadUrl)};
    req.setRawHeader("User-Agent", "PasteTool");
    req.setAttribute(QNetworkRequest::RedirectPolicyAttribute, QNetworkRequest::NoLessSafeRedirectPolicy);

    QEventLoop loop;
    auto *reply = mgr.get(req);
    QObject::connect(reply, &QNetworkReply::finished, &loop, &QEventLoop::quit);
    loop.exec();

    if (reply->error() != QNetworkReply::NoError) {
        reply->deleteLater();
        QMessageBox::critical(nullptr,
            QStringLiteral("错误"),
            QStringLiteral("自动下载失败，请检查网络后重试，或手动打开 latest release 页面下载。"));
        return;
    }

    QFile file(outputPath);
    if (!file.open(QIODevice::WriteOnly)) {
        reply->deleteLater();
        QMessageBox::critical(nullptr,
            QStringLiteral("错误"),
            QStringLiteral("无法写入文件: %1").arg(outputPath));
        return;
    }
    file.write(reply->readAll());
    file.close();
    reply->deleteLater();

    if (launchAfterDownload) {
        int answer = QMessageBox::information(nullptr,
            QStringLiteral("下载完成"),
            QStringLiteral("安装包已下载到：\n%1\n\n是否现在启动安装包？").arg(outputPath),
            QMessageBox::Yes | QMessageBox::No);
        if (answer == QMessageBox::Yes) {
            QDesktopServices::openUrl(QUrl::fromLocalFile(outputPath));
        } else {
            QDesktopServices::openUrl(QUrl::fromLocalFile(downloadDir));
        }
    } else {
        int answer = QMessageBox::information(nullptr,
            QStringLiteral("下载完成"),
            QStringLiteral("最新便携版已下载到：\n%1\n\n是否打开下载目录？").arg(outputPath),
            QMessageBox::Yes | QMessageBox::No);
        if (answer == QMessageBox::Yes) {
            QDesktopServices::openUrl(QUrl::fromLocalFile(downloadDir));
        }
    }
}
