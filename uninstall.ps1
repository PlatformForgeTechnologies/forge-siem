#Requires -RunAsAdministrator
# Forge SIEM Agent — Windows Uninstaller
#
# Usage (PowerShell as Administrator):
#   irm https://platformforge.com/uninstall.ps1 | iex

param(
    [switch]$KeepConfig,
    [string]$InstallDir  = "C:\Program Files\ForgeSIEM",
    [string]$ConfigDir   = "C:\ProgramData\ForgeSIEM\config",
    [string]$StateDir    = "C:\ProgramData\ForgeSIEM\state",
    [string]$ServiceName = "ForgeSIEMAgent"
)

$ErrorActionPreference = "Stop"

function Write-Step { param([string]$msg) Write-Host "[forge-siem] $msg" -ForegroundColor Green }
function Write-Warn { param([string]$msg) Write-Host "[forge-siem] WARNING: $msg" -ForegroundColor Yellow }

# ── Stop and remove service ─────────────────────────────────────────────

$svc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($svc) {
    Write-Step "Stopping service..."
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    Write-Step "Removing service..."
    & sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

# ── Remove binary ───────────────────────────────────────────────────────

$BinaryPath = Join-Path $InstallDir "forge-siem-agent.exe"
if (Test-Path $BinaryPath) {
    Write-Step "Removing binary..."
    Remove-Item $BinaryPath -Force
}
if ((Test-Path $InstallDir) -and -not (Get-ChildItem $InstallDir)) {
    Remove-Item $InstallDir -Force
}

# ── Remove config and state ─────────────────────────────────────────────

if ($KeepConfig) {
    Write-Warn "Keeping config at $ConfigDir and state at $StateDir"
} else {
    if (Test-Path $ConfigDir) {
        Write-Step "Removing config directory..."
        Remove-Item $ConfigDir -Recurse -Force
    }
    if (Test-Path $StateDir) {
        Write-Step "Removing state directory (includes TLS certs)..."
        Remove-Item $StateDir -Recurse -Force
    }
    $parent = "C:\ProgramData\ForgeSIEM"
    if ((Test-Path $parent) -and -not (Get-ChildItem $parent)) {
        Remove-Item $parent -Force
    }
}

# ── Remove registry entry ──────────────────────────────────────────────

$regPath = "HKLM:\SYSTEM\CurrentControlSet\Services\$ServiceName"
if (Test-Path $regPath) {
    Remove-Item $regPath -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "  [OK] Forge SIEM Agent uninstalled" -ForegroundColor Green
if ($KeepConfig) {
    Write-Host "  Config and state preserved."
}
Write-Host ""
