# Antigravity Status Line

A high-performance status line renderer and background billing daemon that tracks local LLM token usage and costs for the Antigravity CLI.

## Features

* **High Performance**: Sub-millisecond Go renderer handles real-time state parsing.
* **Local-First Architecture**: Operates 100% locally with zero dependencies on GCP Cloud Monitoring or active network authentication. No `gcloud` logins or credentials are required.
* **Lock-Free Isolated Logging**: Sessions log token metrics concurrently to isolated, lock-free files (`usage_<conversation_id>_<YYYY-MM-DD>.jsonl`) to prevent terminal blockages and resource contention.
* **Background Daemon**: A minutely scheduler aggregates today's multi-session logs and applies model pricing rules to build daily consumption tables.
* **Responsive Layouts**: Scales across wide, standard, compact, and minimal terminal width breakpoints.
* **Dynamic Warning Indicators**: Automatically displays warnings like `[Daemon Dead]` if background logging stops for more than 5 minutes, or `[Daemon Err]` for systemic exceptions.

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
| `make build-local` | Builds `statusline` and `agy-statusline-daemon` in the repository root. |
| `make build-current` | Builds binaries with matching platform suffix under `releases/`. |
| `make install` | Compiles current platform binaries and runs the installer. |
| `make build-releases`| Cross-compiles optimized binaries for Darwin and Linux (AMD64/ARM64). |
| `make test` | Runs the Go test suite. |
| `make clean` | Removes generated binaries and releases. |

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
