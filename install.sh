#!/usr/bin/env sh
# Forge SIEM Agent — Universal Installer
# Supports Linux (amd64, arm64, armv7) and macOS (amd64, arm64).
#
# Usage:
#   curl -fsSL https://platformforgegroup.com/install.sh | sudo ENROLLMENT_TOKEN=enr_xxx bash
#
# Environment variables:
#   ENROLLMENT_TOKEN (required) Enrollment token from Forge SIEM Settings → Enrollment
#   INGEST_URL       (optional) Override ingest endpoint (default: ingest.platformforgegroup.com)
#   AGENT_VERSION  (optional)  Specific version to install (default: latest)
#   INSTALL_DIR    (optional)  Binary install path (default: /usr/local/bin)
#   SKIP_SERVICE   (optional)  Set to "1" to skip service registration
#
set -eu

AGENT_VERSION="${AGENT_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
CONFIG_DIR="/etc/forge-siem-agent"
STATE_DIR="/var/lib/forge-siem-agent"
SERVICE_NAME="forge-siem-agent"
GITHUB_ORG="PlatformForgeTechnologies"
GITHUB_REPO="forge-siem-agent"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

log()  { printf "${GREEN}[forge-siem]${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}[forge-siem]${NC} %s\n" "$*"; }
die()  { printf "${RED}[forge-siem] ERROR:${NC} %s\n" "$*" >&2; exit 1; }

# ── Service installers ──────────────────────────────────────────────────

install_systemd_service() {
  if ! command -v systemctl >/dev/null 2>&1; then
    warn "systemd not found — skipping service registration. Start manually:"
    warn "  AGENT_CONFIG_PATH=${CONFIG_DIR}/agent.yaml ${INSTALL_DIR}/${SERVICE_NAME}"
    return
  fi

  log "Installing systemd service..."
  cat > "/etc/systemd/system/${SERVICE_NAME}.service" <<UNIT
[Unit]
Description=Forge SIEM Agent
Documentation=https://platformforgegroup.com/docs
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=${INSTALL_DIR}/${SERVICE_NAME}
Restart=on-failure
RestartSec=5s
Environment=AGENT_CONFIG_PATH=${CONFIG_DIR}/agent.yaml

StateDirectory=forge-siem-agent
ConfigurationDirectory=forge-siem-agent

NoNewPrivileges=yes
ProtectSystem=strict
ReadWritePaths=${STATE_DIR}
ReadOnlyPaths=/var/log /proc
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
UNIT

  systemctl daemon-reload
  systemctl enable --now "$SERVICE_NAME"
  log "Service started."
}

install_launchd_service() {
  PLIST="/Library/LaunchDaemons/com.platformforge.siem-agent.plist"

  log "Installing launchd service..."
  cat > "$PLIST" <<PLISTEOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.platformforge.siem-agent</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${SERVICE_NAME}</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>AGENT_CONFIG_PATH</key>
        <string>${CONFIG_DIR}/agent.yaml</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <dict>
        <key>SuccessfulExit</key>
        <false/>
    </dict>
    <key>StandardOutPath</key>
    <string>/var/log/forge-siem-agent.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/forge-siem-agent.err</string>
    <key>ThrottleInterval</key>
    <integer>5</integer>
</dict>
</plist>
PLISTEOF

  launchctl load -w "$PLIST"
  log "Service started."
}

# ── Preflight checks ────────────────────────────────────────────────────

[ "$(id -u)" -ne 0 ] && die "Run as root: curl -fsSL https://platformforgegroup.com/install.sh | sudo ENROLLMENT_TOKEN=enr_xxx bash"

# Accept ENROLLMENT_TOKEN (preferred) or legacy ENROLL_TOKEN.
ENROLL_TOKEN="${ENROLLMENT_TOKEN:-${ENROLL_TOKEN:-}}"
[ -z "$ENROLL_TOKEN" ] && die "ENROLLMENT_TOKEN is required. Generate one from Settings → Enrollment in Forge SIEM."

# Default ingest endpoint.
INGEST_URL="${INGEST_URL:-https://ingest.platformforgegroup.com}"

command -v curl >/dev/null 2>&1 || die "curl is required but not installed"

# ── Detect OS and architecture ───────────────────────────────────────────

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)  OS_SLUG="linux" ;;
  Darwin) OS_SLUG="darwin" ;;
  *)      die "Unsupported OS: $OS (only Linux and macOS are supported)" ;;
esac

case "$ARCH" in
  x86_64|amd64)   ARCH_SLUG="amd64" ;;
  aarch64|arm64)   ARCH_SLUG="arm64" ;;
  armv7l)          ARCH_SLUG="armv7" ;;
  *)               die "Unsupported architecture: $ARCH" ;;
esac

PLATFORM="${OS_SLUG}_${ARCH_SLUG}"

# ── Resolve version ─────────────────────────────────────────────────────

if [ "$AGENT_VERSION" = "latest" ]; then
  log "Resolving latest release..."
  AGENT_VERSION=$(curl -fsSL "https://api.github.com/repos/${GITHUB_ORG}/${GITHUB_REPO}/releases/latest" \
    | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')
  [ -z "$AGENT_VERSION" ] && die "Could not resolve latest version from GitHub"
fi

BINARY_URL="https://github.com/${GITHUB_ORG}/${GITHUB_REPO}/releases/download/v${AGENT_VERSION}/forge-siem-agent_${PLATFORM}"

# ── Banner ───────────────────────────────────────────────────────────────

printf "\n"
printf "  ${BOLD}${CYAN}Forge SIEM Agent Installer${NC}\n"
printf "  ─────────────────────────────\n"
printf "  Version:  ${BOLD}v%s${NC}\n" "$AGENT_VERSION"
printf "  Platform: ${BOLD}%s${NC}\n" "$PLATFORM"
printf "  Endpoint: ${BOLD}%s${NC}\n" "$INGEST_URL"
printf "\n"

# ── Download binary ──────────────────────────────────────────────────────

log "Downloading agent binary..."
TMP_BIN=$(mktemp)
HTTP_CODE=$(curl -fsSL -w "%{http_code}" -o "$TMP_BIN" "$BINARY_URL" 2>/dev/null) || true

if [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "302" ]; then
  rm -f "$TMP_BIN"
  die "Download failed (HTTP $HTTP_CODE). Check that v${AGENT_VERSION} exists for ${PLATFORM}"
fi

# ── Verify binary ───────────────────────────────────────────────────────

SIZE=$(wc -c < "$TMP_BIN" | tr -d ' ')
if [ "$SIZE" -lt 1000 ]; then
  rm -f "$TMP_BIN"
  die "Downloaded file is too small (${SIZE} bytes) — likely not a valid binary"
fi

# ── Install binary ──────────────────────────────────────────────────────

mkdir -p "$INSTALL_DIR"
mv "$TMP_BIN" "${INSTALL_DIR}/${SERVICE_NAME}"
chmod +x "${INSTALL_DIR}/${SERVICE_NAME}"
log "Binary installed to ${INSTALL_DIR}/${SERVICE_NAME}"

# ── Create directories ──────────────────────────────────────────────────

mkdir -p "$CONFIG_DIR" "$STATE_DIR/tls"
chmod 700 "$STATE_DIR"

# ── Write config ────────────────────────────────────────────────────────

if [ ! -f "${CONFIG_DIR}/agent.yaml" ]; then
  log "Writing default config..."
  cat > "${CONFIG_DIR}/agent.yaml" <<AGENTCFG
enrollment:
  url: "${INGEST_URL}"
  token: "${ENROLL_TOKEN}"
  cert_path: ${STATE_DIR}/tls/agent.crt
  key_path:  ${STATE_DIR}/tls/agent.key
  ca_path:   ${STATE_DIR}/tls/ca.crt

server:
  host: ""
  port: 1514
  ca_cert:     ${STATE_DIR}/tls/ca.crt
  client_cert: ${STATE_DIR}/tls/agent.crt
  client_key:  ${STATE_DIR}/tls/agent.key

state:
  path: ${STATE_DIR}/state.json

log_collection:
  paths:
    - /var/log/auth.log
    - /var/log/syslog
    - /var/log/messages
    - /var/log/secure
    - /var/log/audit/audit.log
  outputs:
    siem:
      enabled: true

fim:
  paths:
    - /etc
    - /usr/bin
    - /usr/sbin
  schedule: realtime

process_monitor:
  enabled: true
  interval: 30s

inventory:
  enabled: true
  interval: 6h

response:
  enabled: false
AGENTCFG
  chmod 600 "${CONFIG_DIR}/agent.yaml"
else
  warn "Config already exists at ${CONFIG_DIR}/agent.yaml — skipping (upgrade mode)."
fi

# ── Register service ────────────────────────────────────────────────────

if [ "${SKIP_SERVICE:-0}" = "1" ]; then
  log "Skipping service registration (SKIP_SERVICE=1)"
else
  case "$OS_SLUG" in
    linux)  install_systemd_service ;;
    darwin) install_launchd_service ;;
  esac
fi

# ── Done ────────────────────────────────────────────────────────────────

printf "\n"
printf "  ${BOLD}${GREEN}✓ Forge SIEM Agent installed${NC}\n"
printf "  ─────────────────────────────\n"
printf "  Binary:  %s/%s\n" "$INSTALL_DIR" "$SERVICE_NAME"
printf "  Config:  %s/agent.yaml\n" "$CONFIG_DIR"
printf "  State:   %s/\n" "$STATE_DIR"

case "$OS_SLUG" in
  linux)
    printf "  Logs:    journalctl -u %s -f\n" "$SERVICE_NAME"
    printf "  Stop:    systemctl stop %s\n" "$SERVICE_NAME"
    printf "  Status:  systemctl status %s\n" "$SERVICE_NAME"
    ;;
  darwin)
    printf "  Logs:    tail -f /var/log/forge-siem-agent.log\n"
    printf "  Stop:    launchctl unload /Library/LaunchDaemons/com.platformforge.siem-agent.plist\n"
    printf "  Status:  launchctl list | grep siem-agent\n"
    ;;
esac

printf "\n"
printf "  The agent will self-enroll on first start. Monitor enrollment:\n"
case "$OS_SLUG" in
  linux)  printf "    journalctl -u %s -f | grep enroll\n" "$SERVICE_NAME" ;;
  darwin) printf "    tail -f /var/log/forge-siem-agent.log | grep enroll\n" ;;
esac
printf "\n"
