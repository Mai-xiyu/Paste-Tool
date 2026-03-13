#ifndef MyAppVersion
  #define MyAppVersion "0.0.0"
#endif

#ifndef MyOutputBaseFilename
  #define MyOutputBaseFilename "paste_tool-installer"
#endif

[Setup]
AppId={{B7B962A3-11E4-4E85-88D6-21F5A2ED4F2E}
AppName=Paste Tool
AppVersion={#MyAppVersion}
AppPublisher=Mai-xiyu
DefaultDirName={localappdata}\Programs\Paste Tool
DefaultGroupName=Paste Tool
DisableProgramGroupPage=yes
OutputDir=..\dist
OutputBaseFilename={#MyOutputBaseFilename}
Compression=lzma
SolidCompression=yes
WizardStyle=modern
PrivilegesRequired=lowest
ArchitecturesInstallIn64BitMode=x64compatible
UninstallDisplayIcon={app}\paste_tool.exe

[Files]
Source: "..\dist\paste_tool.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\*.dll"; DestDir: "{app}"; Flags: ignoreversion recursesubdirs
Source: "..\dist\platforms\*"; DestDir: "{app}\platforms"; Flags: ignoreversion recursesubdirs
Source: "..\dist\styles\*"; DestDir: "{app}\styles"; Flags: ignoreversion recursesubdirs
Source: "..\dist\tls\*"; DestDir: "{app}\tls"; Flags: ignoreversion recursesubdirs createallsubdirs

[Icons]
Name: "{group}\Paste Tool"; Filename: "{app}\paste_tool.exe"
Name: "{group}\卸载 Paste Tool"; Filename: "{uninstallexe}"
Name: "{autodesktop}\Paste Tool"; Filename: "{app}\paste_tool.exe"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "创建桌面快捷方式"; GroupDescription: "附加任务："

[Run]
Filename: "{app}\paste_tool.exe"; Description: "安装完成后启动 Paste Tool"; Flags: nowait postinstall skipifsilent
