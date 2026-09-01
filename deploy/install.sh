#!/usr/bin/env bash
# Installs fleet-pulse as a systemd service on a bare-metal Linux host.
# Idempotent: safe to re-run after an upgrade (`make build` then this
# script again) to pick up a new binary.
#
# Usage: sudo ./deploy/install.sh
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
    echo "install.sh must be run as root (try: sudo ./deploy/install.sh)" >&2
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BINARY="${REPO_ROOT}/fleet-pulse"

if [[ ! -f "${BINARY}" ]]; then
    echo "no built binary at ${BINARY} -- run 'make build' first" >&2
    exit 1
fi

INSTALL_DIR="/opt/fleet-pulse"
CONFIG_DIR="/etc/fleet-pulse"

echo "==> creating the fleet-pulse system user/group"
if ! getent group fleet-pulse >/dev/null; then
    groupadd --system fleet-pulse
fi
if ! getent passwd fleet-pulse >/dev/null; then
    useradd --system --no-create-home --shell /usr/sbin/nologin \
        --gid fleet-pulse fleet-pulse
fi

echo "==> installing the binary to ${INSTALL_DIR}"
install -d -m 0755 "${INSTALL_DIR}"
install -m 0755 "${BINARY}" "${INSTALL_DIR}/fleet-pulse"

echo "==> installing config to ${CONFIG_DIR}"
install -d -m 0755 "${CONFIG_DIR}"
if [[ ! -f "${CONFIG_DIR}/config.yaml" ]]; then
    install -m 0644 "${REPO_ROOT}/config.example.yaml" "${CONFIG_DIR}/config.yaml"
    echo "    wrote a default config.yaml -- edit ${CONFIG_DIR}/config.yaml to tune it"
else
    echo "    ${CONFIG_DIR}/config.yaml already exists, leaving it alone"
fi
chown -R fleet-pulse:fleet-pulse "${CONFIG_DIR}"

echo "==> installing the systemd unit"
install -m 0644 "${SCRIPT_DIR}/fleet-pulse.service" /etc/systemd/system/fleet-pulse.service

echo "==> reloading systemd and enabling the service"
systemctl daemon-reload
systemctl enable fleet-pulse.service
systemctl restart fleet-pulse.service

echo "==> done. Check status with: systemctl status fleet-pulse"
