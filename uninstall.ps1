# uninstall.ps1 — Uninstall coder CLI on Windows
#
# Usage:
#   .\uninstall.ps1 -ClearData
#   .\uninstall.ps1 -KeepData

param(
    [switch]$ClearData = $false,
    [switch]$KeepData = $false
)

$Binary = "coder"
$InstallDir = "$env:LOCALAPPDATA\Programs\coder"
$DataDir = "$env:USERPROFILE\.coder"

# Default behavior: keep data if no switch is provided
if ($KeepData) { $ClearData = $false }

$Dest = Get-Command $Binary -ErrorAction SilentlyContinue | Select-Object -ExpandProperty Source
if (-not $Dest) {
    $Dest = Join-Path $InstallDir "$Binary.exe"
}

if (Test-Path $Dest) {
    Write-Host "Uninstalling $Binary from $Dest..."
    Remove-Item -Path $Dest -Force
    Write-Host "✓ Binary removed."
} else {
    Write-Host "⚠ $Binary not found. It might already be uninstalled."
}

# ── Ollama Cleanup ────────────────────────────────────────────────────────────

Write-Host "Cleaning up Ollama services..."

$startupFolder = [System.IO.Path]::Combine($env:APPDATA, "Microsoft\Windows\Start Menu\Programs\Startup")
$shortcutPath = [System.IO.Path]::Combine($startupFolder, "Ollama.lnk")

if (Test-Path $shortcutPath) {
    Write-Host "Removing Ollama startup shortcut..."
    Remove-Item -Path $shortcutPath -Force
    Write-Host "✓ Startup shortcut removed."
}

# Kill running ollama process
$ollamaProc = Get-Process -Name "ollama" -ErrorAction SilentlyContinue
if ($ollamaProc) {
    Write-Host "Terminating running Ollama processes..."
    Stop-Process -Name "ollama" -Force -ErrorAction SilentlyContinue
    Write-Host "✓ Ollama processes terminated."
}

if ($ClearData) {
    if (Test-Path $DataDir) {
        Write-Host "Clearing local data at $DataDir..."
        Remove-Item -Path $DataDir -Recurse -Force
        Write-Host "✓ Local data cleared."
    } else {
        Write-Host "No local data found at $DataDir."
    }
} else {
    Write-Host "Keeping local data at $DataDir."
}

# Remove from PATH if it was added to $InstallDir
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -like "*$InstallDir*") {
    $NewPath = ($UserPath -split ";" | Where-Object { $_ -ne $InstallDir }) -join ";"
    [Environment]::SetEnvironmentVariable("Path", $NewPath, "User")
    Write-Host "Removed $InstallDir from your PATH."
}

Write-Host ""
Write-Host "✓ $Binary has been uninstalled."
