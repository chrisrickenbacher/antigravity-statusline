#!/usr/bin/env bash

set -euo pipefail

BIN_DIR="${HOME}/.gemini/antigravity-cli/bin"
CACHE_DIR="${HOME}/.gemini/antigravity-cli/cache"
SETTINGS_PATH="${HOME}/.gemini/antigravity-cli/settings.json"

echo "=== Antigravity Status Line Uninstaller ==="

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"

# 1. Disable Daemon Schedulers
if [[ "${OS}" == "darwin" ]]; then
  # Clean up legacy daemon
  OLD_PLIST="${HOME}/Library/LaunchAgents/com.antigravity.statusline-daemon.plist"
  if [[ -f "${OLD_PLIST}" ]]; then
    echo "Stopping and unloading legacy macOS launchd service..."
    launchctl bootout "gui/$(id -u)" "${OLD_PLIST}" 2>/dev/null || true
    rm -f "${OLD_PLIST}"
  fi

  # Clean up new daemon
  PLIST_PATH="${HOME}/Library/LaunchAgents/com.antigravity.agy-statusline-daemon.plist"
  if [[ -f "${PLIST_PATH}" ]]; then
    echo "Stopping and unloading macOS launchd service..."
    launchctl bootout "gui/$(id -u)" "${PLIST_PATH}" 2>/dev/null || true
    rm -f "${PLIST_PATH}"
  fi
elif [[ "${OS}" == "linux" ]]; then
  # Clean up legacy daemon
  OLD_TIMER="${HOME}/.config/systemd/user/antigravity-statusline.timer"
  OLD_SERVICE="${HOME}/.config/systemd/user/antigravity-statusline.service"
  if [[ -f "${OLD_TIMER}" ]]; then
    echo "Stopping and disabling legacy Linux systemd user timers..."
    systemctl --user disable --now antigravity-statusline.timer 2>/dev/null || true
    rm -f "${OLD_TIMER}" "${OLD_SERVICE}"
  fi

  # Clean up new daemon
  TIMER_PATH="${HOME}/.config/systemd/user/antigravity-agy-statusline.timer"
  SERVICE_PATH="${HOME}/.config/systemd/user/antigravity-agy-statusline.service"
  if [[ -f "${TIMER_PATH}" ]]; then
    echo "Stopping and disabling Linux systemd user timers..."
    systemctl --user disable --now antigravity-agy-statusline.timer 2>/dev/null || true
    rm -f "${TIMER_PATH}" "${SERVICE_PATH}"
  fi

  systemctl --user daemon-reload
fi

# 2. Remove settings.json Hook
if [[ -f "${SETTINGS_PATH}" ]]; then
  echo "Deconfiguring statusline hook from settings.json..."
  python3 -c "
import json, os
path = '${SETTINGS_PATH}'
if os.path.exists(path):
    try:
        with open(path, 'r') as f:
            settings = json.load(f)
        if 'statusline' in settings:
            del settings['statusline']
        with open(path, 'w') as f:
            json.dump(settings, f, indent=2)
    except Exception as e:
        print('Warning: Failed to deconfigure settings.json:', e)
"
fi

# 3. Wipe File Footprints
echo "Removing binary files..."
rm -f "${BIN_DIR}/statusline" "${BIN_DIR}/statusline-daemon" "${BIN_DIR}/agy-statusline-daemon"

# 4. Optional Cache Purge Prompt (default to safe no-deletion if non-interactive)
PURGE="n"
if [[ -t 0 ]]; then
  read -p "Would you like to completely delete the local metrics and pricing cache files inside '${CACHE_DIR}'? [y/N]: " PURGE
fi

if [[ "${PURGE}" =~ ^[Yy]$ ]]; then
  echo "Purging metrics and pricing cache..."
  rm -f "${CACHE_DIR}/api_usage.json" "${CACHE_DIR}/pricing_cache.json" "${CACHE_DIR}/daemon-err.log" "${CACHE_DIR}/daemon-out.log"
  # remove directory only if empty
  rmdir "${CACHE_DIR}" 2>/dev/null || true
else
  echo "Cache files preserved."
fi

echo "=== Uninstallation Complete! ==="
