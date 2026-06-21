#Requires -RunAsAdministrator
# Forge SIEM Agent — Windows Installer
#
# Usage (PowerShell as Administrator):
#   $env:ENROLL_TOKEN="tok_xxx"; $env:INGEST_URL="https://siem-api.example.com"; irm https://platformforge.com/install.ps1 | iex
#
# Or with explicit params:
#   .\install.ps1 -EnrollToken "tok_xxx" -IngestUrl "https://siem-api.example.com"

param(
    [string]$EnrollToken = $env:ENROLL_TOKEN,
    [string]$IngestUrl   = $env:INGEST_URL,
    [string]$Version     = $env:AGENT_VERSION,
    [string]$InstallDir  = "C:\Program Files\ForgeSIEM",
    [string]$ConfigDir   = "C:\ProgramData\ForgeSIEM\config",
    [string]$StateDir    = "C:\ProgramData\ForgeSIEM\state",
    [string]$ServiceName = "ForgeSIEMAgent"
)

$ErrorActionPreference = "Stop"

$GithubOrg  = "PlatformForgeTechnologies"
$GithubRepo = "forge-siem-agent"

function Write-Step  { param([string]$msg) Write-Host "[forge-siem] $msg" -ForegroundColor Green }
function Write-Warn  { param([string]$msg) Write-Host "[forge-siem] WARNING: $msg" -ForegroundColor Yellow }
function Write-Err   { param([string]$msg) Write-Host "[forge-siem] ERROR: $msg" -ForegroundColor Red }

# ── Preflight ────────────────────────────────────────────────────────────

if (-not $EnrollToken) { throw "ENROLL_TOKEN is required. Create one via: POST /api/v1/enrollment/tokens" }
if (-not $IngestUrl)   { throw "INGEST_URL is required (e.g. https://siem-api.example.com)" }

# ── Resolve version ─────────────────────────────────────────────────────

if (-not $Version -or $Version -eq "latest") {
    Write-Step "Resolving latest version..."
    $rel = Invoke-RestMethod "https://api.github.com/repos/$GithubOrg/$GithubRepo/releases/latest"
    $Version = $rel.tag_name.TrimStart("v")
}

$Arch = if ([Environment]::Is64BitOperatingSystem) { "windows_amd64" } else { "windows_386" }
$BinaryUrl  = "https://github.com/$GithubOrg/$GithubRepo/releases/download/v${Version}/forge-siem-agent_${Arch}.exe"
$BinaryPath = Join-Path $InstallDir "forge-siem-agent.exe"
$ConfigPath = Join-Path $ConfigDir "agent.yaml"

# ── Banner ───────────────────────────────────────────────────────────────

Write-Host ""
Write-Host "  Forge SIEM Agent Installer" -ForegroundColor Cyan
Write-Host "  ----------------------------"
Write-Host "  Version:  v$Version"
Write-Host "  Platform: $Arch"
Write-Host "  Endpoint: $IngestUrl"
Write-Host ""

# ── Create directories ──────────────────────────────────────────────────

foreach ($dir in @($InstallDir, $ConfigDir, $StateDir, (Join-Path $StateDir "tls"))) {
    if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Path $dir -Force | Out-Null }
}

# ── Download binary ─────────────────────────────────────────────────────

Write-Step "Downloading agent binary..."
try {
    Invoke-WebRequest -Uri $BinaryUrl -OutFile $BinaryPath -UseBasicParsing
} catch {
    throw "Download failed from $BinaryUrl — check that v${Version} exists for $Arch"
}

$size = (Get-Item $BinaryPath).Length
if ($size -lt 1000) {
    Remove-Item $BinaryPath
    throw "Downloaded file too small ($size bytes) — likely not a valid binary"
}

Write-Step "Binary installed to $BinaryPath"

# ── Write config ────────────────────────────────────────────────────────

if (-not (Test-Path $ConfigPath)) {
    Write-Step "Writing agent config..."
    @"
enrollment:
  url: "$IngestUrl"
  token: $EnrollToken
  cert_path: $StateDir\tls\agent.crt
  key_path:  $StateDir\tls\agent.key
  ca_path:   $StateDir\tls\ca.crt

server:
  host: ""
  port: 1514
  ca_cert:     $StateDir\tls\ca.crt
  client_cert: $StateDir\tls\agent.crt
  client_key:  $StateDir\tls\agent.key

state:
  path: $StateDir\state.json

log_collection:
  paths:
    - C:\Windows\System32\winevt\Logs\Security.evtx
    - C:\Windows\System32\winevt\Logs\System.evtx
    - C:\Windows\System32\winevt\Logs\Application.evtx
    - C:\inetpub\logs\LogFiles\*\*.log
  outputs:
    siem:
      enabled: true

fim:
  paths:
    - C:\Windows\System32
    - C:\Program Files
  schedule: 5m

process_monitor:
  enabled: true
  interval: 30s

inventory:
  enabled: true
  interval: 6h

response:
  enabled: false
"@ | Set-Content -Path $ConfigPath -Encoding UTF8
} else {
    Write-Warn "Config already exists at $ConfigPath — skipping (upgrade mode)."
}

# ── Register Windows Service ────────────────────────────────────────────

$existingSvc = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
if ($existingSvc) {
    Write-Step "Stopping existing service..."
    Stop-Service -Name $ServiceName -Force -ErrorAction SilentlyContinue
    & sc.exe delete $ServiceName | Out-Null
    Start-Sleep -Seconds 2
}

Write-Step "Registering Windows Service..."
$binPathArg = "`"$BinaryPath`""
& sc.exe create $ServiceName binpath= $binPathArg start= auto DisplayName= "Forge SIEM Agent" | Out-Null
& sc.exe description $ServiceName "Forge SIEM Agent — host telemetry and security monitoring" | Out-Null
& sc.exe failure $ServiceName reset= 60 actions= restart/5000/restart/10000/restart/30000 | Out-Null

# Set AGENT_CONFIG_PATH for the service
& reg add "HKLM\SYSTEM\CurrentControlSet\Services\$ServiceName" /v Environment /t REG_MULTI_SZ /d "AGENT_CONFIG_PATH=$ConfigPath" /f | Out-Null

Write-Step "Starting service..."
Start-Service -Name $ServiceName

$svc = Get-Service -Name $ServiceName

Write-Host ""
Write-Host "  [OK] Forge SIEM Agent installed" -ForegroundColor Green
Write-Host "  ----------------------------"
Write-Host "  Binary:  $BinaryPath"
Write-Host "  Config:  $ConfigPath"
Write-Host "  State:   $StateDir\"
Write-Host "  Service: $($svc.Status)"
Write-Host ""
Write-Host "  Logs:    Get-WinEvent -LogName Application -MaxEvents 50 | Where Source -eq ForgeSIEMAgent"
Write-Host "  Stop:    Stop-Service $ServiceName"
Write-Host "  Remove:  sc.exe delete $ServiceName"
Write-Host ""
Write-Host "  The agent will self-enroll on first start."
Write-Host ""
