#Requires -Version 5.1
[CmdletBinding()]
param(
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\monocron",
    [string]$Repo = "Omotolani98/monocron",
    [string]$App = "monocronctl"
)

$ErrorActionPreference = "Stop"

$arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
$os = "windows"

$release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
$tag = $release.tag_name
if (-not $tag) {
    throw "Failed to fetch latest release."
}

$url = "https://github.com/$Repo/releases/download/$tag/${App}_${tag}_${os}_${arch}.zip"
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

    $currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($currentPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$currentPath;$InstallDir", "User")
        Write-Host "Added $InstallDir to your user PATH. Restart your terminal to use $App."
    }

    Write-Host "$App installed to $InstallDir"
    & (Join-Path $InstallDir "$App.exe") --help | Out-Null
    Write-Host "$App is ready."
} finally {
    Remove-Item -Recurse -Force $tmp -ErrorAction SilentlyContinue
}
