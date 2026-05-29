[Functional Specification] Vibe Port Manager (VPM) - Cross-Platform

This document defines the final functional specifications and operational design of the Vibe Port Manager (VPM), an ultra-lightweight, cross-platform port and process management tool designed to optimize the developer experience for "vibe coders" across Windows, macOS, and Linux.

1. Architecture & System Constraints

Target Platforms: Windows 10/11 (64-bit), macOS (Intel/Apple Silicon), Linux (Ubuntu/Debian-based major distros)

Tech Stack: Go (Wails v2) + HTML/JS (Tailwind CSS)

Performance Targets:

Executable Size: ~10MB to 15MB (Single self-contained native binary)

Idle Memory (RAM) Usage: Under 30MB

Port Consumption: 0 ports (No local loopback web server; communicates via OS-native IPC message loops to bridge UI and Go Backend)

2. Core Challenges & Implementation Specifications

To ensure high performance and prevent technical issues across multiple operating systems, the following business rules and backend behaviors are strictly enforced:

2.1 [Environment Variable Protection] OS-Specific Shell Wrappers

Requirement: Must fully recognize local Node.js/Bun environments managed via tools like NVM, FNM, ASDF, or MISE, inheriting all shell configuration settings.

Implementation Logic:

Windows: Command execution is routed through PowerShell to bypass native shell virtualization sandboxes:

powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "cmd /c <User_Command>"


macOS & Linux: VPM detects the user's default shell (typically /bin/zsh or /bin/bash) and spawns the child process as a Login Shell (-l or --login). This forces the shell to source startup scripts (such as .zshrc, .bash_profile, .profile), seamlessly inheriting dynamically injected environment variables (like PATH modifications for FNM/NVM):

/bin/zsh -l -c "<User_Command>"


2.2 [Anti-Zombie Process] OS-Specific Triple-Pass Port Killer

Requirement: Ensure all nested child processes (e.g., Vite, Next.js Compiler, Nodemon) are completely terminated when a process is stopped, across all platforms.

Implementation Logic:

Phase 1 (Graceful Terminate):

Windows: Issues a process-group termination signal using taskkill /T /PID <PID>.

macOS/Linux: Spawns child processes in their own Process Group (using syscall.Setpgid). Terminations are initiated by sending SIGTERM to the negative Process Group ID (-PGID), shutting down the entire process tree.

Phase 2 (Port Polling): After sending the signal, the Go backend polls the target port status (using native netstat commands or checking port bindings) 3 times at 500ms intervals.

Phase 3 (Forced Terminate Fallback): If the port remains occupied after 1.5 seconds:

Windows: Resolves the blocking PID and forcibly kills it using taskkill /F /PID <Target_PID>.

macOS/Linux: Runs lsof -t -i :<Port> to resolve the exact PID and forcibly kills it via kill -9 <PID>.

2.3 [UI Performance Guard] Log Buffering & Throttler

Requirement: Prevent UI freezing during high-frequency console output.

Implementation Logic:

A 100ms sliding buffer window is applied to the real-time log transmission stream.

Logs are buffered in a Go slice (array) and sent as a single batched array every 100ms.

The frontend UI renderer appends the log batch to the terminal viewport in a single rendering tick, drastically reducing CPU overhead.

2.4 [System Tray Minimization] Minimize-to-Tray & Background Persistence

Requirement: Ensure development processes remain active when the application window is closed or minimized, with quick-toggle controls from the system tray.

Implementation Logic:

VPM leverages Wails' built-in system tray and menu features.

Minimize/Close Behavior: When the user clicks the "Close (X)" button, the application window is hidden instead of terminated, moving to the system tray (Windows Taskbar Tray, macOS Menu Bar, or Linux System Tray).

Dynamic Context Menu: The system tray icon provides a right-click context menu that dynamically lists all registered workspaces, showing their current status (Green/Gray) and offering a quick toggle option: [Start] / [Stop] for each workspace, along with an [Open UI] and [Quit VPM] button.

2.5 [Persistent Logging & Export] OS-Specific Log Rotation & Archiving

Requirement: Write console output streams to disk for error tracking and support log exports.

Implementation Logic:

All stdout/stderr output is concurrently streamed to a local log file structured by workspace and date:

Windows: %APPDATA%/vpm/logs/<workspace_id>/<YYYY-MM-DD>.log

macOS: ~/Library/Application Support/vpm/logs/<workspace_id>/<YYYY-MM-DD>.log

Linux: ~/.config/vpm/logs/<workspace_id>/<YYYY-MM-DD>.log

VPM maintains a maximum of 7 days of logs per workspace, automatically purging older entries (log rotation).

The frontend features an [Export Logs] button, which triggers a Wails native Save File Dialog to let users save a consolidated .txt or .zip log package to their desktop.

2.6 [Smart Port Auto-Reassign] Intelligent Port Recommendation Wizard

Requirement: Suggest alternative free ports when a collision occurs, rather than just forcing a termination.

Implementation Logic:

When VPM detects a port collision during startup, it initiates an incremental background scan starting from the target port (e.g., Target_Port + 1, + 2...) to find the nearest unoccupied port.

Instead of immediately triggering a blocking error, the UI displays a smart wizard dialog:

Modal: "Port <Target_Port> is currently occupied. Would you like to automatically switch to free port <Recommended_Port> instead?"

If accepted, VPM updates the active runtime environment variable configuration (PORT=<Recommended_Port>) and launches the process seamlessly.

3. UI/UX Interface Specifications (Tabs & Layout)

VPM features a modern, responsive, dark-themed viewport divided into three main tabs:

Tab 1: Workspace Register

Purpose: Register and map target directories to command scripts and ports.

UI Components:

[Directory Selector & Drag-and-Drop Area]: Interactive zone to bind local project folders.

[Smart Package Manager Detection]:

Once a directory is selected, the Go backend scans for Lock Files:

yarn.lock -> Automatically pre-selects and configures Yarn package manager commands (e.g., yarn dev).

pnpm-lock.yaml -> Pre-selects pnpm (e.g., pnpm dev).

bun.lockb -> Pre-selects Bun (e.g., bun dev).

package-lock.json or fallback -> Defaults to npm (e.g., npm run dev).

Parses package.json scripts block to populate a quick-selection dropdown list.

[Custom Override Fields]: Allows manual override for completely custom command scripts (e.g., python -m http.server, go run main.go) and port assignments.

[Save & Activate Button]: Serializes configurations to %APPDATA%/vpm/config.json (or OS equivalent).

Tab 2: Port Dashboard

Purpose: Monitor system-wide port allocations and manage active workspaces.

UI Components:

[App Control Cards Grid]:

Displays Workspace Name, Port, Package Manager badge, and status indicators: RUNNING (Green), STOPPED (Gray), PORT COLLISION (Orange).

Control Actions: [Start], [Restart], and [Stop].

[Smart Port Switch Wizard]: Prompts when a port collision occurs, offering to either [Force Kill Blocking Process] or [Run on Free Port XXXX].

[System Port Analyzer Panel]: Scans and displays a live list of occupied system ports and their corresponding PIDs, featuring a [Port Killer] action.

Tab 3: Log Live Monitor

Purpose: Real-time console inspection.

UI Components:

[Console Log Terminal View]: Dark terminal-inspired output with locked auto-scrolling utilizing monospaced fonts.

[Terminal Toolbar]:

[Auto-Scroll Lock Toggle]

[Copy Logs to Clipboard] (Direct mapping to OS Clipboard)

[Export Logs] (Triggers native file saving for persistent log files)

[Clear View]

4. Build & Distribution Rules

To compile standalone binaries for multiple platforms:

System Prerequisites: Go SDK (1.21+), Node.js (18+), MSVC Build Tools (Windows), Xcode CLI Tools (macOS), or GCC (Linux).

Build Commands:

Windows:

wails build -clean -platform windows/amd64


macOS:

wails build -clean -platform darwin/universal


Linux:

wails build -clean -platform linux/amd64


Output Artifact: A zero-dependency, self-contained executable optimized for the respective platform, with integrated system tray assets and an embedded frontend.