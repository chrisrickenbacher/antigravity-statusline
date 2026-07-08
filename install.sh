#!/usr/bin/env bash

set -euo pipefail

BIN_DIR="${HOME}/.gemini/antigravity-cli/bin"
CACHE_DIR="${HOME}/.gemini/antigravity-cli/cache"
SETTINGS_PATH="${HOME}/.gemini/antigravity-cli/settings.json"
REPO_URL="https://raw.githubusercontent.com/chrisrickenbacher/antigravity-statusline/main"

echo "=== Antigravity Status Line Installer ==="

# 1. Environment & Platform Resolution
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "${OS}" in
  darwin)
    PLATFORM_OS="darwin"
    ;;
  linux)
    PLATFORM_OS="linux"
    ;;
  *)
    echo "Error: Unsupported Operating System: ${OS}" >&2
    exit 1
    ;;
esac

case "${ARCH}" in
  x86_64|amd64)
    PLATFORM_ARCH="amd64"
    ;;
  arm64|aarch64)
    PLATFORM_ARCH="arm64"
    ;;
  *)
    echo "Error: Unsupported Architecture: ${ARCH}" >&2
    exit 1
    ;;
esac

TARGET_SUFFIX="${PLATFORM_OS}-${PLATFORM_ARCH}"
echo "Detected platform: ${TARGET_SUFFIX}"

# 2. Setup Directories
mkdir -p "${BIN_DIR}" "${CACHE_DIR}"

# 3. Precompiled Asset Fetching & Atomic Overwrite
if [[ -f "./releases/statusline-${TARGET_SUFFIX}" && -f "./releases/statusline-daemon-${TARGET_SUFFIX}" ]]; then
  echo "Found local compiled assets in releases/. Installing directly..."
  cp "./releases/statusline-${TARGET_SUFFIX}" "${BIN_DIR}/statusline.tmp"
  cp "./releases/statusline-daemon-${TARGET_SUFFIX}" "${BIN_DIR}/statusline-daemon.tmp"
else
  STATUSLINE_URL="${REPO_URL}/releases/statusline-${TARGET_SUFFIX}"
  DAEMON_URL="${REPO_URL}/releases/statusline-daemon-${TARGET_SUFFIX}"

  echo "Fetching precompiled statusline binary from GitHub..."
  curl -fsSL -o "${BIN_DIR}/statusline.tmp" "${STATUSLINE_URL}"

  echo "Fetching precompiled billing daemon binary from GitHub..."
  curl -fsSL -o "${BIN_DIR}/statusline-daemon.tmp" "${DAEMON_URL}"
fi

mv "${BIN_DIR}/statusline.tmp" "${BIN_DIR}/statusline"
mv "${BIN_DIR}/statusline-daemon.tmp" "${BIN_DIR}/statusline-daemon"
chmod +x "${BIN_DIR}/statusline" "${BIN_DIR}/statusline-daemon"

echo "Binaries safely installed in ${BIN_DIR}"

# 4. Daemon Scheduler Configuration & Reloading
GCP_ENV=""
if [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]]; then
  GCP_ENV="GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS}"
fi

PROJECT_ENV=""
if [[ -n "${GCP_PROJECT_ID:-}" ]]; then
  PROJECT_ENV="GCP_PROJECT_ID=${GCP_PROJECT_ID}"
elif [[ -n "${GOOGLE_CLOUD_PROJECT:-}" ]]; then
  PROJECT_ENV="GCP_PROJECT_ID=${GOOGLE_CLOUD_PROJECT}"
fi

if [[ "${PLATFORM_OS}" == "darwin" ]]; then
  PLIST_DIR="${HOME}/Library/LaunchAgents"
  PLIST_PATH="${PLIST_DIR}/com.antigravity.statusline-daemon.plist"
  mkdir -p "${PLIST_DIR}"

  echo "Configuring macOS launchd service..."

  ENV_XML=""
  if [[ -n "${GCP_ENV}" || -n "${PROJECT_ENV}" ]]; then
    ENV_XML="<key>EnvironmentVariables</key><dict>"
    if [[ -n "${GCP_ENV}" ]]; then
      ENV_XML="${ENV_XML}<key>GOOGLE_APPLICATION_CREDENTIALS</key><string>${GOOGLE_APPLICATION_CREDENTIALS}</string>"
    fi
    if [[ -n "${PROJECT_ENV}" ]]; then
      VAL="${GCP_PROJECT_ID:-${GOOGLE_CLOUD_PROJECT}}"
      ENV_XML="${ENV_XML}<key>GCP_PROJECT_ID</key><string>${VAL}</string>"
    fi
    ENV_XML="${ENV_XML}</dict>"
  fi

  cat <<EOF > "${PLIST_PATH}"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.antigravity.statusline-daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>${BIN_DIR}/statusline-daemon</string>
    </array>
    <key>StartInterval</key>
    <integer>300</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>${CACHE_DIR}/daemon-err.log</string>
    <key>StandardOutPath</key>
    <string>${CACHE_DIR}/daemon-out.log</string>
    ${ENV_XML}
</dict>
</plist>
EOF

  echo "Reloading LaunchAgent..."
  launchctl bootout "gui/$(id -u)" "${PLIST_PATH}" 2>/dev/null || true
  launchctl bootstrap "gui/$(id -u)" "${PLIST_PATH}"

elif [[ "${PLATFORM_OS}" == "linux" ]]; then
  SYSTEMD_DIR="${HOME}/.config/systemd/user"
  mkdir -p "${SYSTEMD_DIR}"

  echo "Configuring Linux systemd user service..."

  SERVICE_ENV=""
  if [[ -n "${GCP_ENV}" ]]; then
    SERVICE_ENV="Environment=\"GOOGLE_APPLICATION_CREDENTIALS=${GOOGLE_APPLICATION_CREDENTIALS}\""
  fi
  if [[ -n "${PROJECT_ENV}" ]]; then
    VAL="${GCP_PROJECT_ID:-${GOOGLE_CLOUD_PROJECT}}"
    SERVICE_ENV="${SERVICE_ENV}
Environment=\"GCP_PROJECT_ID=${VAL}\""
  fi

  cat <<EOF > "${SYSTEMD_DIR}/antigravity-statusline.service"
[Unit]
Description=Antigravity Statusline Billing Daemon
After=network.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/statusline-daemon
${SERVICE_ENV}

[Install]
WantedBy=default.target
EOF

  cat <<EOF > "${SYSTEMD_DIR}/antigravity-statusline.timer"
[Unit]
Description=Run Antigravity Statusline Billing Daemon every 5 minutes

[Timer]
OnCalendar=*:0/5
Persistent=true

[Install]
WantedBy=timers.target
EOF

  echo "Reloading systemd user configs..."
  systemctl --user daemon-reload
  systemctl --user enable --now antigravity-statusline.timer
  systemctl --user restart antigravity-statusline.timer
fi

# 5. CLI Settings Integration
echo "Integrating status hook with settings.json..."
python3 -c "
import json, os
path = '${SETTINGS_PATH}'
settings = {}
if os.path.exists(path):
    try:
        with open(path, 'r') as f:
            settings = json.load(f)
    except Exception:
        pass

settings['statusline'] = {
    'enabled': True,
    'command': '${BIN_DIR}/statusline'
}

os.makedirs(os.path.dirname(path), exist_ok=True)
with open(path, 'w') as f:
    json.dump(settings, f, indent=2)
"

echo "=== Installation & Integration Successful! ==="
