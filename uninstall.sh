#!/usr/bin/env sh
# Forge SIEM Agent — Universal Uninstaller
#
# Usage:
#   curl -fsSL https://platformforge.com/uninstall.sh | sudo sh
#
# Environment variables:
#   KEEP_CONFIG  (optional)  Set to "1" to keep config and state directories
#
set -eu

SERVICE_NAME="forge-siem-agent"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/forge-siem-agent"
STATE_DIR="/var/lib/forge-siem-agent"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log()  { printf "${GREEN}[forge-siem]${NC} %s\n" "$*"; }
warn() { printf "${YELLOW}[forge-siem]${NC} %s\n" "$*"; }
die()  { printf "${RED}[forge-siem] ERROR:${NC} %s\n" "$*" >&2; exit 1; }

[ "$(id -u)" -ne 0 ] && die "Run as root: curl -fsSL ... | sudo sh"

OS="$(uname -s)"

# ── Stop and remove service ─────────────────────────────────────────────

case "$OS" in
  Linux)
    if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet "$SERVICE_NAME" 2>/dev/null; then
      log "Stopping systemd service..."
      systemctl stop "$SERVICE_NAME"
      systemctl disable "$SERVICE_NAME" 2>/dev/null || true
    fi
    if [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
      log "Removing systemd unit..."
      rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
      systemctl daemon-reload
    fi
    ;;
  Darwin)
    PLIST="/Library/LaunchDaemons/com.platformforge.siem-agent.plist"
    if [ -f "$PLIST" ]; then
      log "Unloading launchd service..."
      launchctl unload -w "$PLIST" 2>/dev/null || true
      rm -f "$PLIST"
    fi
    ;;
esac

# ── Remove binary ───────────────────────────────────────────────────────

if [ -f "${INSTALL_DIR}/${SERVICE_NAME}" ]; then
  log "Removing binary..."
  rm -f "${INSTALL_DIR}/${SERVICE_NAME}"
fi

# ── Remove config and state ─────────────────────────────────────────────

if [ "${KEEP_CONFIG:-0}" = "1" ]; then
  warn "Keeping config at ${CONFIG_DIR} and state at ${STATE_DIR} (KEEP_CONFIG=1)"
else
  if [ -d "$CONFIG_DIR" ]; then
    log "Removing config directory..."
    rm -rf "$CONFIG_DIR"
  fi
  if [ -d "$STATE_DIR" ]; then
    log "Removing state directory (includes TLS certs)..."
    rm -rf "$STATE_DIR"
  fi
fi

# ── Clean up macOS logs ─────────────────────────────────────────────────

if [ "$OS" = "Darwin" ]; then
  rm -f /var/log/forge-siem-agent.log /var/log/forge-siem-agent.err 2>/dev/null || true
fi

printf "\n"
printf "  ${GREEN}✓ Forge SIEM Agent uninstalled${NC}\n"
if [ "${KEEP_CONFIG:-0}" = "1" ]; then
  printf "  Config and state preserved at %s and %s\n" "$CONFIG_DIR" "$STATE_DIR"
fi
printf "\n"
