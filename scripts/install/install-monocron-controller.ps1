#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\monocron",
    [string]$Repo = "Omotolani98/monocron",
    [string]$App = "monocron-controller"
)

$ErrorActionPreference = "Stop"

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$os = "windows"

$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$tag = $release.tag_name
if (-not $tag) {
    throw "Failed to fetch latest release."
}

$url = "https://github.com/$Repo/releases/download/$tag/${App}_$($tag.TrimStart('v'))_${os}_${arch}.zip"
$tmp = Join-Path $env:TEMP ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tmp | Out-Null

try {
    Write-Host "Downloading $App $tag for $os/$arch..."
    $zip = Join-Path $tmp "$App.zip"
    Invoke-RestMethod -Uri $url -OutFile $zip

    Write-Host "Extracting..."
    Expand-Archive -Path $zip -DestinationPath $tmp -Force

    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $exe = Join-Path $tmp "$App.exe"
    Move-Item -Path $exe -Destination (Join-Path $InstallDir "$App.exe") -Force

    Write-Host "$App installed to $InstallDir"
    Write-Host "Set MONOCRON_DATABASE_URL and run the controller:"
    Write-Host "  `$env:MONOCRON_DATABASE_URL = 'postgres://user:pass@localhost/monocron?sslmode=disable'"
    Write-Host "  $InstallDir\$App.exe"
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
