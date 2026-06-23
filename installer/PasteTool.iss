#define MyAppName "Paste Tool"
#define MyAppPublisher "Mai-xiyu"
#define MyAppURL "https://github.com/Mai-xiyu/Paste-Tool"
#define MyAppVersion GetEnv("PASTE_TOOL_VERSION")
#define SourceExe GetEnv("PASTE_TOOL_SOURCE_EXE")
#define OutputDir GetEnv("PASTE_TOOL_OUTPUT_DIR")
#define OutputBaseName GetEnv("PASTE_TOOL_INSTALLER_BASENAME")

[Setup]
AppId={{5A491D0B-7C37-4B4D-A1C6-26B85FDE4B3F}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
AppPublisherURL={#MyAppURL}
AppSupportURL={#MyAppURL}
AppUpdatesURL={#MyAppURL}/releases/latest
DefaultDirName={autopf}\Paste Tool
DefaultGroupName=Paste Tool
DisableProgramGroupPage=yes
OutputDir={#OutputDir}
OutputBaseFilename={#OutputBaseName}
Compression=lzma
SolidCompression=yes
WizardStyle=modern
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "Create a desktop shortcut"; GroupDescription: "Additional icons:"; Flags: unchecked

[Files]
Source: "{#SourceExe}"; DestDir: "{app}"; DestName: "paste_tool.exe"; Flags: ignoreversion

[Icons]
Name: "{group}\Paste Tool"; Filename: "{app}\paste_tool.exe"
Name: "{autodesktop}\Paste Tool"; Filename: "{app}\paste_tool.exe"; Tasks: desktopicon

[Run]
Filename: "{app}\paste_tool.exe"; Description: "Launch Paste Tool"; Flags: nowait postinstall skipifsilent
