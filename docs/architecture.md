# Architecture Diagram

This document illustrates the design and pipeline of the Antigravity Status Line rendering engine and background billing daemon.

```mermaid
graph TD
    AG["Antigravity CLI"] -->|Stdin state stream| SL["statusline CLI Renderer"]
    SL -->|Renders formatted statusline| AG
    
    Daemon["agy-statusline-daemon Background Job"] -->|5 min poll| GCP["GCP Cloud Monitoring"]
    Daemon -->|JSON Cache| CacheDir[("~/.gemini/antigravity-cli/cache/")]
    CacheDir -->|Read| SL
```

## Description

1. **State Parsing (`statusline`)**: The CLI renderer reads structured snake_case payloads from stdin, computes turn/session costs using local caches, and prints formatted output to stdout in $<2\text{ms}$.
2. **Billing Daemon (`agy-statusline-daemon`)**: A background service (configured via systemd or launchd) that runs periodically, queries GCP Monitoring for active Vertex AI usage metrics, fetches model pricing rates, and atomically caches the data as JSON files.
