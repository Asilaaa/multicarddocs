#!/usr/bin/env bash
set -euo pipefail

APP_NAME="multicard-mcp-go"
SERVICE_NAME="multicard-mcp"
INSTALL_DIR="${INSTALL_DIR:-/opt/multicard-mcp}"
CONFIG_DIR="${CONFIG_DIR:-/etc/multicard-mcp}"
SYSTEMD_DIR="${SYSTEMD_DIR:-/etc/systemd/system}"
RUN_USER="${RUN_USER:-multicard-mcp}"
RUN_GROUP="${RUN_GROUP:-multicard-mcp}"
SUDO_CMD="${SUDO_CMD:-sudo}"

if ! command -v "$SUDO_CMD" >/dev/null 2>&1; then
  SUDO_CMD=""
fi

run() {
  if [ -n "$SUDO_CMD" ]; then
    "$SUDO_CMD" "$@"
  else
    "$@"
  fi
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if ! getent group "$RUN_GROUP" >/dev/null 2>&1; then
  run groupadd --system "$RUN_GROUP"
fi
if ! id -u "$RUN_USER" >/dev/null 2>&1; then
  run useradd --system --gid "$RUN_GROUP" --home-dir "$INSTALL_DIR" --shell /usr/sbin/nologin "$RUN_USER"
fi

mkdir -p bin
if [ ! -x "bin/$APP_NAME" ]; then
  echo "[install] local binary not found in bin/$APP_NAME, building it"
  go build -o "bin/$APP_NAME" .
fi

run mkdir -p "$INSTALL_DIR/bin" "$CONFIG_DIR"
run install -m 0755 "bin/$APP_NAME" "$INSTALL_DIR/bin/$APP_NAME"
run rm -rf "$INSTALL_DIR/multicard-docs"
run cp -a multicard-docs "$INSTALL_DIR/"
run install -m 0644 deploy/systemd/${SERVICE_NAME}.service "$SYSTEMD_DIR/${SERVICE_NAME}.service"

if [ ! -f "$CONFIG_DIR/${SERVICE_NAME}.env" ]; then
  run install -m 0644 deploy/systemd/${SERVICE_NAME}.env.example "$CONFIG_DIR/${SERVICE_NAME}.env"
fi

run chown -R root:root "$INSTALL_DIR"
run chown -R "$RUN_USER:$RUN_GROUP" "$INSTALL_DIR/multicard-docs"
run systemctl daemon-reload

echo "[install] installed to $INSTALL_DIR"
echo "[install] edit $CONFIG_DIR/${SERVICE_NAME}.env if needed"
echo "[install] then run: ${SUDO_CMD:+sudo }systemctl enable --now ${SERVICE_NAME}.service"
