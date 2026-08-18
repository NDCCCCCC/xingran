#!/usr/bin/env bash
# setup-server.sh — one-time server bootstrap for XingRan deployment.
#
# Run ON THE TARGET SERVER as user `ubuntu` (passwordless sudo, the Tencent
# Lighthouse default) BEFORE the first CI deploy:
#
#   1. creates /opt/xingran directory layout (configs/logs/uploads/data/deploy)
#   2. creates /etc/xingran/secrets.env skeleton (600 root:root) if absent
#   3. verifies /opt/xingran/configs/config.yaml exists (must be copied from
#      configs/config.prod.example.yaml and edited manually — sqlite mode)
#
# Idempotent: safe to re-run; existing files are never overwritten.
set -euo pipefail

APP=/opt/xingran

echo "==> creating $APP layout"
sudo mkdir -p "$APP"/{configs,logs,uploads,data,deploy}
sudo chown -R ubuntu:ubuntu "$APP"

echo "==> secrets skeleton"
if [ ! -f /etc/xingran/secrets.env ]; then
  sudo mkdir -p /etc/xingran
  sudo tee /etc/xingran/secrets.env >/dev/null <<'EOF'
# XingRan secrets — injected by systemd (pid 1 reads this, 600 root:root).
# Fill real values, then: sudo systemctl restart xingran
# JWT_ACCESS_SECRET / JWT_REFRESH_SECRET: openssl rand -base64 48
# SM4_KEY: openssl rand -base64 16
JWT_ACCESS_SECRET=change-me
JWT_REFRESH_SECRET=change-me
SM4_KEY=
EOF
  sudo chmod 600 /etc/xingran/secrets.env
  sudo chown root:root /etc/xingran/secrets.env
  echo "    created /etc/xingran/secrets.env — EDIT IT before first deploy"
else
  echo "    /etc/xingran/secrets.env exists, leaving untouched"
fi

echo "==> config check"
if [ -f "$APP/configs/config.yaml" ]; then
  echo "    $APP/configs/config.yaml exists"
else
  cat >&2 <<EOF
!!
!! MISSING $APP/configs/config.yaml
!! Copy it from the repo template and edit (sqlite mode, port 9000, release):
!!   scp configs/config.prod.example.yaml ubuntu@<host>:$APP/configs/config.yaml
!!   vim $APP/configs/config.yaml
!!
EOF
fi

echo "==> sudo sanity"
sudo -n true && echo "    passwordless sudo OK" || echo "    WARNING: passwordless sudo unavailable — deploy will fail"

echo "==> done. Next: add the CI deploy public key to ~/.ssh/authorized_keys"
