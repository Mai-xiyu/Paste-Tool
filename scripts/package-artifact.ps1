param(
    [Parameter(Mandatory = $true)]
    [string]$VersionLabel,

    [string]$DistDirectory = "dist",

    [string]$ExecutablePath = "dist/paste_tool.exe",

    [string]$ReadmePath = "README.md"
)

if (-not (Test-Path -Path $ExecutablePath)) {
    Write-Error "Missing executable: $ExecutablePath"
    exit 1
}

if (-not (Test-Path -Path $ReadmePath)) {
    Write-Error "Missing README: $ReadmePath"
    exit 1
}

New-Item -ItemType Directory -Path $DistDirectory -Force | Out-Null

$packageRoot = Join-Path $DistDirectory "package"
$archiveName = "paste_tool-$VersionLabel-windows-x64.zip"
$archivePath = Join-Path $DistDirectory $archiveName
$versionFilePath = Join-Path $packageRoot "VERSION.txt"

if (Test-Path -Path $packageRoot) {
    Remove-Item -Path $packageRoot -Recurse -Force
}

New-Item -ItemType Directory -Path $packageRoot -Force | Out-Null
Copy-Item -Path $ExecutablePath -Destination (Join-Path $packageRoot "paste_tool.exe") -Force
Copy-Item -Path $ReadmePath -Destination (Join-Path $packageRoot "README.md") -Force

@(
    "Version: $VersionLabel"
    "Latest Release: https://github.com/Mai-xiyu/Paste-Tool/releases/latest"
    "Repository: https://github.com/Mai-xiyu/Paste-Tool"
) | Set-Content -Path $versionFilePath -Encoding UTF8

if (Test-Path -Path $archivePath) {
    Remove-Item -Path $archivePath -Force
}

Compress-Archive -Path (Join-Path $packageRoot "*") -DestinationPath $archivePath -Force

Write-Output $archivePath
