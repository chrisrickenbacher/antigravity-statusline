# Debugging Guide

This guide explains how to troubleshoot, inspect, and debug the `statusline` renderer and the background `statusline-daemon` service.

## 1. Background Daemon Logs

The daemon logs stdout and stderr differently depending on your operating system.

### macOS (launchd)
The daemon is managed as a user LaunchAgent. Its logs are redirected to files inside the cache directory:
* **Error Log**: `~/.gemini/antigravity-cli/cache/daemon-err.log`
* **Output Log**: `~/.gemini/antigravity-cli/cache/daemon-out.log`

To watch logs in real-time:
```bash
tail -f ~/.gemini/antigravity-cli/cache/daemon-err.log
```

### Linux (systemd)
The daemon runs as a systemd user service. Its logs are collected by the standard systemd journal:
* **View Logs**:
  ```bash
  journalctl --user -u antigravity-statusline -n 50 --no-pager
  ```
* **Watch Logs (Follow)**:
  ```bash
  journalctl --user -u antigravity-statusline -f
  ```

---

## 2. Interactive / Manual Execution

You can run both binaries directly from the terminal to bypass schedulers and view direct stdout/stderr outputs.

### Debugging the Daemon Interactively
Run the binary directly. It will attempt to poll GCP Monitoring and fetch pricing, printing any warnings or errors immediately to stderr:
```bash
~/.gemini/antigravity-cli/bin/statusline-daemon
```

### Debugging the Status Line Renderer
The renderer accepts a JSON payload over stdin. You can feed it a mock payload using `echo` or `cat` to inspect exact visual outputs:

```bash
echo '{
  "agent_state": "thinking",
  "model": {
    "id": "gemini-3.5-flash",
    "display_name": "Gemini 3.5 Flash"
  },
  "context_window": {
    "current_usage": {
      "input_tokens": 1200,
      "output_tokens": 350
    },
    "total_input_tokens": 25000,
    "total_output_tokens": 5000
  },
  "terminal_width": 100
}' | ~/.gemini/antigravity-cli/bin/statusline
```

---

## 3. Cache Directory Inspection

The renderer and daemon communicate through JSON state files stored inside:
`~/.gemini/antigravity-cli/cache/`

Ensure these files exist, are valid JSON, and check their internal metadata for errors:
* **`api_usage.json`**: Contains today's API billing stats and any GCP monitoring `network_error` or `auth_error` messages.
* **`pricing_cache.json`**: Stores cached model pricing configurations.
