# Vibe Port Manager (VPM)

**A polyglot dev-server port manager for Windows.** Start, stop, restart, and monitor multiple development servers across different projects and languages — all from one unified dark-themed desktop app.

## Features

### 🚀 Multi-Language Support
- **JavaScript/TypeScript**: Single apps (npm/yarn/pnpm/bun), monorepo workspaces, PM2 ecosystem configs
- **Backend**: Python (Django, FastAPI, Flask), Go, Rust
- **.NET**: ASP.NET Core with launchSettings port extraction
- **Ruby**: Rails, Rack apps
- **PHP**: Laravel, Symfony, built-in server
- **Java**: Spring Boot (Maven/Gradle with wrapper detection)
- **Other**: Elixir/Phoenix, Deno, Docker Compose, Procfile, Make/Just/Taskfile

### ⚡ Smart Detection
- **Auto-detect** dev servers from project files (package.json, go.mod, Cargo.toml, etc.)
- **Monorepo workspaces** — npm/yarn/pnpm/bun workspaces with per-app port extraction
- **PM2 ecosystem** — import multiple servers from ecosystem.config.js without PM2 runtime
- **Docker Compose** — parse services and expose host ports
- **Fallback mode** — if no primary detector matches, try Make/Just/Taskfile dev targets

### 💾 Port Management
- **Graceful shutdown** — SIGTERM → polling → force kill (Windows taskkill /T)
- **Port reclamation** — resolve orphaned processes (e.g. npm→cmd→node trees) and retry
- **Free port suggestion** — one-click reassign when collision detected
- **Port re-detection** — Re-detect button to track servers that drifted from configured port

### 👁️ UI/UX
- **Dense server list** — scannable rows (status · port · name · command · uptime) with hover actions
- **Mini mode** — always-on-top floating bar showing active port status; click a chip to expand
- **Frameless dark UI** — custom titlebar, unified gray-950/900/800 theme with indigo accents
- **Integrated log strip** — collapsible at bottom, searchable, export to file
- **System Ports panel** — scan all listening ports, bulk kill by port number

### 🔧 Advanced
- **Crash auto-restart** — optionally auto-restart servers on unexpected exit (with crash-loop detection)
- **Headless daemon mode** — `vpm --daemon` for CI/CD or boot persistence via registry
- **Per-user storage** — config in `%APPDATA%\vpm\config.json`, logs in `%APPDATA%\vpm\logs\`
- **Atomic config saves** — temp→rename pattern, no corruption on power loss
- **Log rotation** — 7-day retention, 100MB cap per project, rotate on startup + hourly

## Installation

### From Release
1. Download `vibe-port-manager.exe` from [GitHub Releases](https://github.com/simiroa/VibePortManager/releases)
2. Run the executable — no installer needed, all data goes to `%APPDATA%\vpm\`

### From Source
```bash
git clone https://github.com/simiroa/VibePortManager.git
cd VibePortManager
wails build -platform windows -output vibe-port-manager.exe
```

**Requirements:**
- Windows 10 / 11
- Go 1.21+
- Node.js 18+ (for building frontend)
- Wails v2

## Quick Start

1. **Launch VPM** — click the executable or add to Windows autostart
2. **Add a project** — click "+ Add Project" in the sidebar, point to your project directory
3. **VPM auto-detects** dev servers:
   - ✅ If a single dev script or ecosystem is found → ready to go
   - ❓ If multiple or ambiguous → import dialog lets you choose + edit ports/commands
4. **Start a server** — click the row or use Start button
5. **Watch the log** — logs appear in the collapsible strip at bottom
6. **Mini mode** — press `▬` to collapse to a floating port-status bar

## Configuration

**Config file:** `%APPDATA%\vpm\config.json`

```json
{
  "projects": [
    {
      "id": "banana-app",
      "name": "Bananadancer",
      "path": "C:\\Users\\dev\\Projects\\bananadancer",
      "execution_target": "windows-native",
      "servers": [
        {
          "id": "srv-1",
          "name": "frontend",
          "port": 5183,
          "command": "npm run dev -- --port 5183 --strictPort",
          "autostart": true,
          "autorestart": true
        }
      ]
    }
  ]
}
```

**Fields:**
- `execution_target` — `"windows-native"` (only current support)
- `autostart` — start this server when VPM launches
- `autorestart` — auto-restart on crash (max 5 in 60s window)

## Development

### Build Wails app
```bash
wails dev
```
Opens a live-reload dev window with hot-module reload for frontend.

### Run tests
```bash
go test ./... -timeout=30s
go vet ./...
go run ./scripts/validate-specs.go  # Cross-layer SSOT check
```

## Limitations

| Feature | Status | Notes |
|---|---|---|
| macOS / Linux build | ❌ | Windows-only MVP |
| Per-server tray menu | ❌ | Tray only has Show/Quit |
| Logs for externally-started servers | ❌ | OS constraint: can't attach to foreign stdout |
| Command palette | ❌ | Future enhancement |

## Performance

- **Exe size**: ~10 MB (Wails + WebView2)
- **Idle RAM**: ~66 MB (Wails baseline)
- **Listening ports**: 0 (app doesn't bind; queries system)

## License

[License TBD — see LICENSE file]

## Contributing

Contributions welcome! Areas for help:
- macOS/Linux builds
- Additional language detectors
- UI/UX enhancements
- Performance optimizations

---

**Built with [Wails](https://wails.io) + Go + Vanilla JavaScript**
