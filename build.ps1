# Build script for Desktop Pet.
# Produces dist\desktop-pet.exe. Uses the GUI subsystem (no console window).
param(
    [switch]$Debug
)

$ErrorActionPreference = "Stop"

# --- locate the Go toolchain ------------------------------------------------
$go = $null
if (Get-Command go -ErrorAction SilentlyContinue) { $go = (Get-Command go).Source }
elseif (Test-Path "$env:LOCALAPPDATA\Programs\Go\bin\go.exe") { $go = "$env:LOCALAPPDATA\Programs\Go\bin\go.exe" }
elseif (Test-Path "C:\Program Files\Go\bin\go.exe") { $go = "C:\Program Files\Go\bin\go.exe" }
elseif (Test-Path "C:\Program Files\FlyEnv-Data\app\static-go-1.26.6\bin\go.exe") { $go = "C:\Program Files\FlyEnv-Data\app\static-go-1.26.6\bin\go.exe" }
if (-not $go) { Write-Error "Go toolchain not found. Install Go or edit the path in build.ps1." }
Write-Host "Using Go: $go"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

# --- regenerate the built-in sprite PNGs ------------------------------------
& $go run ./cmd/genassets
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

# --- build -------------------------------------------------------------------
New-Item -ItemType Directory -Path (Join-Path $root "dist") -Force | Out-Null
$trim = "-trimpath"
$ldflags = "-s -w"
if (-not $Debug) { $ldflags += " -H=windowsgui" }
$out = Join-Path $root "dist\desktop-pet.exe"

Write-Host "Building $out ..."
& $go build $trim -ldflags $ldflags -o $out ./cmd/desktop-pet
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$size = [math]::Round((Get-Item $out).Length / 1MB, 2)
Write-Host "Done: $out ($size MB)"
