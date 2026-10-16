#!/usr/bin/env bash
# deploy-remote.sh — runs ON THE TARGET SERVER (via ssh 'bash -s').
#
# Activates a new xingran-backend binary previously scp'd to
# /opt/xingran/xingran-backend.new:
#
#   1. idempotently install the systemd unit (skip if unchanged)
#   2. back up the current binary (keep last 5)
#   3. atomically replace + systemctl restart
#   4. health-check /health for up to 60s, verifying the version string
#   5. on failure: roll back to the newest backup, restart, dump journal, exit 1
#
# Usage: deploy-remote.sh <version>     (version = ldflags-injected main.Version)
set -euo pipefail

APP=/opt/xingran
BIN=$APP/xingran-backend
NEW=$BIN.new
UNIT_NEW=$APP/deploy/xingran.service.new
UNIT=/etc/systemd/system/xingran.service
SERVICE=xingran
KEEP=5
HEALTH_URL=http://127.0.0.1:9000/health
VERSION="${1:-}"

fail() {
  echo "!! $*" >&2
  exit 1
}

rollback() {
  local last_bak
  last_bak=$(ls -1t "$BIN".bak-* 2>/dev/null | head -1 || true)
  if [ -n "$last_bak" ]; then
    echo "!! health check failed — rolling back to $last_bak"
    sudo install -m 0755 "$last_bak" "$BIN"
    sudo systemctl restart "$SERVICE"
  else
    echo "!! health check failed and no backup exists — service left in current state"
  fi
  echo "--- last 80 journal lines for $SERVICE ---"
  sudo journalctl -u "$SERVICE" -n 80 --no-pager || true
  exit 1
}

[ -f "$NEW" ] || fail "missing $NEW (was the scp step skipped?)"
[ -n "$VERSION" ] || fail "usage: $0 <version>"

# 1. Idempotent systemd unit install (only on drift)
if ! sudo cmp -s "$UNIT_NEW" "$UNIT" 2>/dev/null; then
  echo "installing systemd unit (changed or first deploy)"
  sudo install -m 644 "$UNIT_NEW" "$UNIT"
  sudo systemctl daemon-reload
  sudo systemctl enable "$SERVICE"
else
  echo "systemd unit unchanged, skipping daemon-reload"
fi

# 2. Back up current binary (keep newest $KEEP)
if [ -f "$BIN" ]; then
  cp -a "$BIN" "$BIN.bak-$(date +%Y%m%d-%H%M%S)"
  ls -1t "$BIN".bak-* 2>/dev/null | tail -n +"$((KEEP + 1))" | xargs -r rm -f
fi

# 3. Atomic replace + restart
sudo install -m 0755 "$NEW" "$BIN"
sudo systemctl restart "$SERVICE"

# 4. Health check: 30 attempts x 2s; on success verify deployed version
for _ in $(seq 1 30); do
  if body=$(curl -fsS "$HEALTH_URL" 2>/dev/null); then
    if echo "$body" | grep -q "\"version\":\"$VERSION\""; then
      echo "deployed $VERSION OK: $body"
      exit 0
    fi
    # Service is up but serving an old version — binary swap did not take.
    echo "version mismatch (wanted $VERSION): $body"
    rollback
  fi
  sleep 2
done

rollback
