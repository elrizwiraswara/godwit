; Windows installer for Godwit (desktop app + bundled engine + model).
; Built by scripts/build_windows.ps1 after "flutter build windows --release".

#ifndef AppVersion
  #define AppVersion "1.0.0"
#endif
#define AppName "Godwit"
#define AppExe "godwit.exe"
#define BuildDir "..\..\app\build\windows\x64\runner\Release"

[Setup]
; Keep AppId the same across versions so upgrades replace the old install
AppId={{80F02339-0155-4DBE-908F-8083839C17C7}
AppName={#AppName}
AppVersion={#AppVersion}
AppPublisher=Elriz Technology
DefaultDirName={autopf}\{#AppName}
DefaultGroupName={#AppName}
DisableProgramGroupPage=yes
; Installs for the current user without administrator rights
PrivilegesRequired=lowest
PrivilegesRequiredOverridesAllowed=dialog
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
OutputDir=..\..\dist
OutputBaseFilename=Godwit-Setup-{#AppVersion}
Compression=lzma2/max
SolidCompression=yes
WizardStyle=modern
UninstallDisplayIcon={app}\{#AppExe}
SetupIconFile=..\..\app\windows\runner\resources\app_icon.ico
CloseApplications=yes

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "{#BuildDir}\*"; DestDir: "{app}"; Flags: recursesubdirs ignoreversion

[Icons]
Name: "{group}\{#AppName}"; Filename: "{app}\{#AppExe}"
Name: "{autodesktop}\{#AppName}"; Filename: "{app}\{#AppExe}"; Tasks: desktopicon

[Run]
Filename: "{app}\{#AppExe}"; Description: "{cm:LaunchProgram,{#AppName}}"; Flags: nowait postinstall skipifsilent
