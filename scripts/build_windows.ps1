# Builds the Windows desktop app with the bundled Go engine and embedding
# model, then the installer (dist\Godwit-Setup-<version>.exe).
#
# Usage: powershell -ExecutionPolicy Bypass -File scripts\build_windows.ps1 [-Version 1.0.0] [-SkipInstaller]
#
# Requires Go, Flutter with Windows desktop support, and Inno Setup 6.

param(
    [string]$Version = "1.0.0",
    [switch]$SkipInstaller
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "app\backend_dist"

Write-Host "Building the engine..."
if (Test-Path $dist) { Remove-Item -Recurse -Force $dist }
New-Item -ItemType Directory -Force (Join-Path $dist "models") | Out-Null
$env:CGO_ENABLED = "0"
# -H=windowsgui: no console window pops up when the app starts the engine
go build -C $root -trimpath -ldflags "-s -w -H=windowsgui" -o (Join-Path $dist "godwit-engine.exe") ./server
if ($LASTEXITCODE -ne 0) { throw "go build failed" }
Copy-Item -Recurse (Join-Path $root "models\all-MiniLM-L6-v2") (Join-Path $dist "models\")

Write-Host "Building the desktop app..."
Push-Location (Join-Path $root "app")
try {
    # A clean build: an incremental one can keep an icon font subset made before an icon was
    # first used, and that icon then shows as blank
    flutter clean | Out-Null
    flutter build windows --release --build-name $Version
    if ($LASTEXITCODE -ne 0) { throw "flutter build failed" }
} finally {
    Pop-Location
}

if ($SkipInstaller) { return }

Write-Host "Building the installer..."
$iscc = (Get-Command iscc -ErrorAction SilentlyContinue).Source
if (-not $iscc) { $iscc = "${env:ProgramFiles(x86)}\Inno Setup 6\ISCC.exe" }
if (-not (Test-Path $iscc)) { throw "Inno Setup 6 not found; install it from https://jrsoftware.org/isinfo.php" }
& $iscc "/DAppVersion=$Version" (Join-Path $root "installer\windows\godwit.iss")
if ($LASTEXITCODE -ne 0) { throw "installer build failed" }

Write-Host "Done: $(Join-Path $root "dist\Godwit-Setup-$Version.exe")"
