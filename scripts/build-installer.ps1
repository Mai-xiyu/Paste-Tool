param(
    [Parameter(Mandatory = $true)]
    [string]$VersionLabel,

    [string]$DistDirectory = "dist",

    [string]$InnoSetupCompilerPath = "C:\Program Files (x86)\Inno Setup 6\ISCC.exe"
)

if (-not (Test-Path -Path $InnoSetupCompilerPath)) {
    Write-Error "Missing Inno Setup compiler: $InnoSetupCompilerPath"
    exit 1
}

New-Item -ItemType Directory -Path $DistDirectory -Force | Out-Null

$outputBaseFilename = "paste_tool-installer-$VersionLabel-windows-x64"

$arguments = @(
    "/DMyAppVersion=$VersionLabel"
    "/DMyOutputBaseFilename=$outputBaseFilename"
    "installer\PasteTool.iss"
)

& $InnoSetupCompilerPath @arguments

if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Join-Path $DistDirectory "$outputBaseFilename.exe"
