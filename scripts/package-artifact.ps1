param(
    [Parameter(Mandatory = $true)]
    [string]$VersionLabel,

    [string]$DistDirectory = "dist",

    [string]$ExecutablePath = "dist/paste_tool.exe"
)

if (-not (Test-Path -Path $ExecutablePath)) {
    Write-Error "Missing executable: $ExecutablePath"
    exit 1
}

New-Item -ItemType Directory -Path $DistDirectory -Force | Out-Null

$artifactName = "paste_tool-$VersionLabel-windows-x64.exe"
$artifactPath = Join-Path $DistDirectory $artifactName

if (Test-Path -Path $artifactPath) {
    Remove-Item -Path $artifactPath -Force
}

Copy-Item -Path $ExecutablePath -Destination $artifactPath -Force

Write-Output $artifactPath
