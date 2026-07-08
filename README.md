# Antigravity Status Line

A high-performance status line renderer and background billing daemon that tracks Vertex AI token usage and costs for the Antigravity CLI.

## Features

* **High Performance**: Sub-millisecond Go renderer handles real-time state parsing.
* **Background Daemon**: Polls Google Cloud Monitoring metrics and dynamic pricing to cache daily token counts.
* **Responsive Layouts**: Scales across wide, standard, compact, and minimal terminal width breakpoints.
* **Robust Cache**: Implements lock-free atomic file writing to prevent I/O blocking during active terminal sessions.

## Installation

### From Source (Developers)
Builds and installs binaries locally:
```bash
git clone https://github.com/chrisrickenbacher/antigravity-statusline.git
cd antigravity-statusline
make install
```

### From Release Binaries
Fetches precompiled binaries directly:
```bash
curl -fsSL https://raw.githubusercontent.com/chrisrickenbacher/antigravity-statusline/main/install.sh | bash
```

## Makefile Targets

| Command | Action |
| :--- | :--- |
| `make build-local` | Builds `statusline` and `statusline-daemon` in the repository root. |
| `make build-current` | Builds binaries with matching platform suffix under `releases/`. |
| `make install` | Compiles current platform binaries and runs the installer. |
| `make build-releases`| Cross-compiles optimized binaries for Darwin and Linux (AMD64/ARM64). |
| `make test` | Runs the Go test suite. |
| `make clean` | Removes generated binaries and releases. |

## GCP Authentication

To enable background metrics polling, authenticate your session:
```bash
gcloud auth application-default login
```

## Uninstallation

To remove binaries, disable background schedulers, and restore `settings.json`:
```bash
./uninstall.sh
```

## Troubleshooting

See [Debugging Guide](docs/debugging.md) for logs, manual test payloads, and troubleshooting commands.

## Contributing

Contributions are welcome. Please open an issue or submit a pull request for any bugs, features, or layout improvements.

## License

MIT License. See [LICENSE](LICENSE) for details.
