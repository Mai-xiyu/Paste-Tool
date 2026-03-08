param(
    [string]$MetadataPath = "app_metadata.h"
)

$content = Get-Content -Path $MetadataPath -Raw
$match = [regex]::Match($content, '#define\s+APP_VERSION\s+L"([^"]+)"')

if (-not $match.Success) {
    Write-Error "Unable to find APP_VERSION in $MetadataPath"
    exit 1
}

$match.Groups[1].Value
