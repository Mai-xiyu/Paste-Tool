param(
    [Parameter(Mandatory = $true)]
    [string]$VersionLabel,

    [string]$DistDirectory = "dist"
)

if (-not (Test-Path -Path "$DistDirectory/paste_tool.exe")) {
    Write-Error "Missing executable: $DistDirectory/paste_tool.exe"
    exit 1
}

$artifactName = "paste_tool-$VersionLabel-windows-x64.exe"
$artifactPath = Join-Path $DistDirectory $artifactName

if ($artifactName -ne "paste_tool.exe") {
    Copy-Item -Path "$DistDirectory/paste_tool.exe" -Destination $artifactPath -Force
}

Write-Host "Portable artifact ready: $artifactPath"
