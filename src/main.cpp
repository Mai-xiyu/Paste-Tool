#include <QApplication>
#include "PasteApp.h"

int main(int argc, char *argv[]) {
    QApplication app(argc, argv);
    app.setQuitOnLastWindowClosed(false);

    PasteApp pasteApp;
    Q_UNUSED(pasteApp);

    return app.exec();
}
