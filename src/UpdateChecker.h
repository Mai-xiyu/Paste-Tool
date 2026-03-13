#ifndef UPDATE_CHECKER_H
#define UPDATE_CHECKER_H

#include <QString>

class UpdateChecker {
public:
    static void checkForUpdates(const QString &currentVersion, const QString &latestReleaseUrl);
    static void downloadAsset(const QString &downloadUrl, const QString &fileName, bool launchAfterDownload);

private:
    static int compareVersions(const QString &current, const QString &latest);
};

#endif
