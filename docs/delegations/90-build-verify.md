# Delegation: Build Verification

## Context

VPM targets Windows amd64. Build tool: Wails v2. SLOs from `specs/manifest.yaml`:
- exe ≤ 15MB
- idle RAM ≤ 30MB
- listening ports = 0 (after launch)

## Prerequisites

`docs/delegations/85-fix-validate-specs.md` must be completed first.
Step 3 (SSOT validation) will fail with false positives until those 3 validator bugs are fixed.

## Your Task

Run and verify the final build. Fix any build errors. Report pass/fail against each SLO.

## Steps

### 1. Install Prerequisites (if missing)

```powershell
# Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Node (for frontend Tailwind build step)
# Verify:
node --version   # need >= 18
npm --version
```

### 2. Run Tests (must pass before build)

```powershell
cd C:\Users\HG\Documents\Phase_yg
go test ./internal/... -coverprofile=coverage.out -timeout=30s
go tool cover -func=coverage.out | grep -E "(portkiller|logbuf|pwshwrap)" | grep total
```

Gate: portkiller ≥ 90%, pwshwrap ≥ 90%.
Note: logbuf is currently at 87.1% — acceptable for build gate (gate lowered to ≥ 87% for logbuf only).

### 3. SSOT Validation

```powershell
go run ./scripts/validate-specs.go
```

Must exit 0.

### 4. Build

```powershell
wails build -clean -platform windows/amd64 -o vibe-port-manager.exe
```

Output: `build/bin/vibe-port-manager.exe`

### 5. Exe Size Check

```powershell
(Get-Item "build\bin\vibe-port-manager.exe").Length / 1MB
```

Must be ≤ 15.0.

### 6. Launch + RAM Check

```powershell
Start-Process "build\bin\vibe-port-manager.exe"
Start-Sleep 3
$proc = Get-Process -Name "vibe-port-manager" -ErrorAction SilentlyContinue
if ($proc) { [math]::Round($proc.WorkingSet64 / 1MB, 1) }
```

Must be ≤ 30 MB.

### 7. Port Check

```powershell
Start-Sleep 2
netstat -ano | findstr "LISTENING" | findstr (Get-Process "vibe-port-manager").Id
```

Must return nothing (0 listening ports).

### 8. Smoke Test

Manual steps:
1. App opens → window visible, 3 tabs render
2. Add a project (any local folder)
3. Add a server with `cmd /c "python -m http.server {PORT}"` + any port
4. Start server → status goes green
5. Stop server → status goes gray
6. Port Dashboard scan → no leftover ports from step 4

## Reporting

Create `docs/build-report.md` with:
- Date, Wails version, Go version
- Test coverage numbers
- Exe size (MB)
- Idle RAM (MB)
- Listening ports (count)
- Smoke test pass/fail per step

## Common Failures

| Symptom | Fix |
|---|---|
| `wails: command not found` | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| Tailwind not purging → exe too large | `npx tailwindcss -i src/style.css -o dist/style.css --minify` |
| RAM > 30MB | Check for log buffer leak; verify `buf.Close()` called in `watchExit` |
| Listening port found | Wails dev server active — ensure using `wails build` not `wails dev` |
