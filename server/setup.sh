#!/usr/bin/env bash
set -euo pipefail

INSTALL_DIR="/opt/letitrip"

if ! command -v docker >/dev/null 2>&1; then
  curl -fsSL https://get.docker.com | sh
  sudo usermod -aG docker "$USER"
fi

sudo mkdir -p "$INSTALL_DIR"
sudo cp "$(dirname "$0")/docker-compose.yml" "$INSTALL_DIR/docker-compose.yml"

if [ ! -f "$INSTALL_DIR/.env" ]; then
  sudo tee "$INSTALL_DIR/.env" >/dev/null <<'EOF'
ND_SCANSCHEDULE=1h
ND_LOGLEVEL=info
ND_SESSIONTIMEOUT=24h
ND_BASEURL=
EOF
fi

echo "Setup complete. Run: cd $INSTALL_DIR && docker compose up -d"
