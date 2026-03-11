# install.ps1 — Install coder CLI on Windows
#
# Usage (run in PowerShell as Administrator or with -Scope CurrentUser):
#   irm https://raw.githubusercontent.com/hiimtrung/coder/main/install.ps1 | iex
#   & ([scriptblock]::Create((irm 'https://raw.githubusercontent.com/hiimtrung/coder/main/install.ps1'))) -Version v0.1.0
#
param(
    [string]$Version = "",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\coder"
)

$ErrorActionPreference = "Stop"
$Repo    = "hiimtrung/coder"
$Binary  = "coder"
$Asset   = "${Binary}-windows-amd64.exe"

# ── Dependencies ──────────────────────────────────────────────────────────────



# ── Resolve version ────────────────────────────────────────────────────────────
if (-not $Version) {
    Write-Host "Fetching latest release..."
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
    if (-not $Version) {
        Write-Error "Failed to fetch latest version from GitHub."
        exit 1
    }
}

Write-Host "Installing $Binary $Version (windows/amd64)..."

# ── Download ───────────────────────────────────────────────────────────────────
$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$Asset"
$TmpFile     = [System.IO.Path]::GetTempFileName() + ".exe"

Write-Host "Downloading: $DownloadUrl"
try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TmpFile -UseBasicParsing
} catch {
    Write-Error "Download failed. Check that release $Version exists: https://github.com/$Repo/releases"
    exit 1
}

# ── Install ────────────────────────────────────────────────────────────────────
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}

$Dest = Join-Path $InstallDir "${Binary}.exe"
Move-Item -Path $TmpFile -Destination $Dest -Force

# Add to PATH for current user if not already present
$UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($UserPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$UserPath;$InstallDir", "User")
    Write-Host "Added $InstallDir to your PATH (restart your terminal to take effect)"
}

Write-Host ""
Write-Host "✓ Installed: $Dest"
Write-Host ""

# ── Initialize Config ─────────────────────────────────────────────────────────

$ConfigDir = Join-Path $env:USERPROFILE ".coder"
$ConfigFile = Join-Path $ConfigDir "config.json"

if (-not (Test-Path $ConfigDir)) {
    New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
}

if (-not (Test-Path $ConfigFile)) {
    Write-Host "Initializing configuration..."
    
    # Prompt for coder-node protocol and URL
    while ($true) {
        Write-Host ""
        Write-Host "Choose coder-node protocol:"
        Write-Host "  1) gRPC (recommended for performance)"
        Write-Host "  2) HTTP (easier for some firewalls/proxies)"
        $ProtoChoice = Read-Host "Selection [1]"
        
        $Protocol = "grpc"
        $DefaultUrl = "localhost:50051"
        if ($ProtoChoice -eq "2") {
            $Protocol = "http"
            $DefaultUrl = "localhost:8080"
        }

        $NodeUrl = Read-Host "Enter coder-node $Protocol URL [$DefaultUrl]"
        if ([string]::IsNullOrWhiteSpace($NodeUrl)) {
            $NodeUrl = $DefaultUrl
        }
        
        Write-Host "Verifying connection to coder-node ($Protocol) at $NodeUrl..."
        
        $Config = @{
            memory = @{
                provider = "remote"
                protocol = $Protocol
                base_url = $NodeUrl
            }
        }
        $Config | ConvertTo-Json -Depth 10 | Out-File -FilePath $ConfigFile -Encoding UTF8
        
        # Test connection using the installed binary
        $ErrFile = Join-Path $ConfigDir "nodecheck.err"
        $process = Start-Process -FilePath $Dest -ArgumentList "memory list --limit 1" -NoNewWindow -Wait -PassThru -RedirectStandardError $ErrFile -RedirectStandardOutput $null
        
        if ($process.ExitCode -eq 0) {
            Write-Host "✓ connection to coder-node successful."
            Remove-Item -Path $ErrFile -ErrorAction SilentlyContinue
            break
        } else {
            Write-Host "⚠ Failed to connect to coder-node ($Protocol). Error details:"
            Get-Content $ErrFile | Write-Host
            Remove-Item -Path $ErrFile -ErrorAction SilentlyContinue
            $Choice = Read-Host "Do you want to re-enter the configuration? [Y/n]"
            if ($Choice -match "^[nN]") {
                break
            }
        }
    }
}
Write-Host "Get started:"
Write-Host "  $Binary install be        # backend project"
Write-Host "  $Binary install fe        # frontend project"
Write-Host "  $Binary install fullstack # full-stack project"
Write-Host "  $Binary list              # see all options"
