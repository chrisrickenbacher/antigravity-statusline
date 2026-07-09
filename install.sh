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

# 2b. Legacy Cleanup (statusline-daemon)
echo "Checking for legacy configurations..."

OLD_PLIST="${HOME}/Library/LaunchAgents/com.antigravity.statusline-daemon.plist"
if [[ -f "${OLD_PLIST}" ]]; then
  echo "Cleaning up legacy macOS LaunchAgent..."
  launchctl bootout "gui/$(id -u)" "${OLD_PLIST}" 2>/dev/null || true
  rm -f "${OLD_PLIST}"
fi

OLD_SYSTEMD_DIR="${HOME}/.config/systemd/user"
if [[ -f "${OLD_SYSTEMD_DIR}/antigravity-statusline.timer" ]]; then
  echo "Cleaning up legacy Linux systemd timer..."
  systemctl --user disable --now antigravity-statusline.timer 2>/dev/null || true
  rm -f "${OLD_SYSTEMD_DIR}/antigravity-statusline.service" "${OLD_SYSTEMD_DIR}/antigravity-statusline.timer"
  systemctl --user daemon-reload
fi

rm -f "${BIN_DIR}/statusline-daemon"

# 2c. Legacy Log & State Migration
echo "Migrating files and cleaning up obsolete cache artifacts..."
rm -f "${CACHE_DIR}/local_usage_"*.jsonl
rm -f "${CACHE_DIR}/last_logged_turn.json"
rm -f "${CACHE_DIR}/last_logged_turns.json"

# 3. Precompiled Asset Fetching & Atomic Overwrite
if [[ -f "./releases/statusline-${TARGET_SUFFIX}" && -f "./releases/agy-statusline-daemon-${TARGET_SUFFIX}" ]]; then
  echo "Found local compiled assets in releases/. Installing directly..."
  cp "./releases/statusline-${TARGET_SUFFIX}" "${BIN_DIR}/statusline.tmp"
  cp "./releases/agy-statusline-daemon-${TARGET_SUFFIX}" "${BIN_DIR}/agy-statusline-daemon.tmp"
else
  STATUSLINE_URL="${REPO_URL}/releases/statusline-${TARGET_SUFFIX}"
  DAEMON_URL="${REPO_URL}/releases/agy-statusline-daemon-${TARGET_SUFFIX}"

  echo "Fetching precompiled statusline binary from GitHub..."
  curl -fsSL -o "${BIN_DIR}/statusline.tmp" "${STATUSLINE_URL}"

  echo "Fetching precompiled billing daemon binary from GitHub..."
  curl -fsSL -o "${BIN_DIR}/agy-statusline-daemon.tmp" "${DAEMON_URL}"
fi

mv "${BIN_DIR}/statusline.tmp" "${BIN_DIR}/statusline"
mv "${BIN_DIR}/agy-statusline-daemon.tmp" "${BIN_DIR}/agy-statusline-daemon"
chmod +x "${BIN_DIR}/statusline" "${BIN_DIR}/agy-statusline-daemon"

echo "Binaries safely installed in ${BIN_DIR}"

# 4. Daemon Scheduler Configuration & Reloading
if [[ "${PLATFORM_OS}" == "darwin" ]]; then
  PLIST_DIR="${HOME}/Library/LaunchAgents"
  PLIST_PATH="${PLIST_DIR}/com.antigravity.agy-statusline-daemon.plist"
  mkdir -p "${PLIST_DIR}"

  echo "Configuring macOS launchd service..."

  cat <<EOF > "${PLIST_PATH}"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.antigravity.agy-statusline-daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>${BIN_DIR}/agy-statusline-daemon</string>
    </array>
    <key>StartInterval</key>
    <integer>60</integer>
    <key>RunAtLoad</key>
    <true/>
    <key>StandardErrorPath</key>
    <string>${CACHE_DIR}/daemon-err.log</string>
    <key>StandardOutPath</key>
    <string>${CACHE_DIR}/daemon-out.log</string>
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

  cat <<EOF > "${SYSTEMD_DIR}/antigravity-agy-statusline.service"
[Unit]
Description=Antigravity Statusline Billing Daemon
After=network.target

[Service]
Type=simple
ExecStart=${BIN_DIR}/agy-statusline-daemon

[Install]
WantedBy=default.target
EOF

  cat <<EOF > "${SYSTEMD_DIR}/antigravity-agy-statusline.timer"
[Unit]
Description=Run Antigravity Statusline Billing Daemon every 1 minute

[Timer]
OnCalendar=minutely
Persistent=true

[Install]
WantedBy=timers.target
EOF

  echo "Reloading systemd user configs..."
  systemctl --user daemon-reload
  systemctl --user enable --now antigravity-agy-statusline.timer
  systemctl --user restart antigravity-agy-statusline.timer
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

settings['statusLine'] = {
    'enabled': True,
    'command': '${BIN_DIR}/statusline'
}
if 'statusline' in settings:
    settings.pop('statusline')

os.makedirs(os.path.dirname(path), exist_ok=True)
with open(path, 'w') as f:
    json.dump(settings, f, indent=2)
"

echo "=== Installation & Integration Successful! ==="
