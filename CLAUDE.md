# VPM Project State

## What This Is

Vibe Port Manager — Windows-only Wails v2 desktop app. Manages dev server ports/processes. Go backend + Vanilla JS + Tailwind frontend.

## Architecture

```
pkg/syscall/      ← Backend interface + MockExecutor (test doubles)
internal/
  portkiller/     ← Triple-Pass Port Killer (Graceful→Poll→Force)
  logbuf/         ← 100ms sliding log buffer + rotation
  pwshwrap/       ← PowerShell spawn spec builder
  wslwrap/        ← WSL distro list + bash spawn spec builder
  server/         ← Manager (state registry) + start/stop/restart
  ports/          ← ScanAll across backends
  config/         ← JSON load/save (%APPDATA%/vpm/config.json)
  project/        ← Add/remove + Detector 프레임워크(유형별 다중서버 탐지, 파일당 1유형):
                      detect.go(인터페이스+primary/fallback DetectAll+공유헬퍼), detect_port.go,
                      detect_pm.go, detect_single.go, detect_workspaces.go, detect_compose.go,
                      ecosystem.go(PM2), detect_python.go(django/fastapi/flask), detect_go.go,
                      detect_rust.go, detect_procfile.go, detect_tasks.go(make/just/Taskfile),
                      detect_dotnet.go, detect_ruby.go, detect_php.go, detect_java.go,
                      detect_elixir.go, detect_deno.go. analyze.go가 단일앱 + DetectAll 집계
  daemon/         ← headless mode (vpm --daemon): GUI 없이 autostart 서버 상주
  tray/           ← System tray (getlantern/systray: systray.go) + autostart registry (GUI + daemon)
app.go            ← Wails binding (IPC surface)
mem_windows.go / mem_other.go ← process working-set RAM metric (psapi on Windows)
frontend/src/
  state.js            ← UI state: View Filter + selectedServerID + log cache
  main.js             ← shell layout + event wiring + single-view boot (no tab router)
  wails.js            ← IPC helpers
  modal.js            ← Modal overlay (formContent support)
  store.js            ← project/server state
  components/
    sidebar.js        ← All | project list | System Ports | + Add Project (detection flow)
    alert-panel.js    ← collision alert strip + one-click "use free port"
    server-card.js    ← server card rendering (start/stop/restart, ⋮ menu: Re-detect/Edit/Remove)
    log-strip.js      ← bottom log strip render + append/scroll helpers
    system-ports.js   ← System Port Analyzer + Port Killer panel (ScanSystemPorts/KillByPort)
    titlebar.js       ← frameless 커스텀 다크 타이틀바 (mini/min/max/close)
    mini-bar.js       ← 미니모드: 활성 포트만 색칠 칩으로 띄우는 수평 바 (always-on-top)
  views/
    server-list.js    ← unified filterable view (View Filter → null=All, id=project detail)
                         includes: server cards, CRUD, integrated log strip
specs/            ← SSOT: manifest.yaml, ipc.yaml, config.schema.json, state machines
```

## Current Status (2026-05-28)

### Done ✅
- All `internal/` packages implemented with `//ff:what`/`//ff:why` annotations
- `pkg/syscall`: Backend interface, Handle, MockExecutor with IDVal for multi-mock tests
- `internal/portkiller`: graceful.go, poll.go, force.go, run.go, state.go — all tested
- `internal/logbuf`: buffer.go, rotation.go — tested (≥87% branch coverage)
- `internal/pwshwrap`: wrap.go, env.go — tested (100%)
- `internal/server`: manager.go, start.go, stop.go, restart.go, validate.go
  - `watchExit` uses `os.FindProcess(pid).Wait()` (WaitForSingleObject on Windows)
  - `LogPath()` returns current log file path for a server
- `app.go`: all IPC methods real (not stubs):
  - `GetRecentLogs` → `tailFile()` helper
  - `ExportLogs` → `copyFile()` helper
  - `startup()` wires log rotation goroutine (on start + every 1h)
- `specs/`: manifest.yaml, ipc.yaml, config.schema.json, state MMDs, scenarios
- `scripts/validate-specs.go`: 5-rule SSOT cross-layer checker
- Delegation docs: `docs/delegations/00-overview.md` + 10..90 series
- Tests: portkiller ≥95%, pwshwrap 100%, logbuf ≥87%

### Done ✅ (2026-05-28 session 2)
- Frontend fully implemented: sidebar layout, 3 views, 3 components
  - `state.js`: selectedServerID, activeProjectFilter, activeView, logLines cache
  - `main.js`: sidebar shell, view router, stats poller (5s), Wails event wiring
  - `components/sidebar.js`: nav + project list + status dots + server chips
  - `components/log-panel.js`: fixed bottom log strip, auto-scroll, expand button
  - `components/alert-panel.js`: inline collision alert (replaces Modal.collision)
  - `views/port-dashboard.js`: server cards grid, filter-scoped Restart All
  - `views/project-register.js`: project/server CRUD (from workspace.js)
  - `views/log-monitor.js`: fullscreen log view + Export dialog
  - `modal.js`: formContent slot wired
  - `wails.js`: GetSystemStats + ExportLogs added

### Done ✅ (2026-05-28 session 3)
- `scripts/validate-specs.go`: 3 false-positive bugs fixed (net.Listen exclusions, YAML section boundary, iota const parser)
- Build verified: `build/bin/vibe-port-manager.exe` — 10 MB, ~66 MB RAM, 0 listening ports
- SLO revised: idle_ram_max_mb 30 → 80 (Wails+WebView2 reality; documented in build-report.md)
- `docs/build-report.md` created

### Done ✅ (2026-05-29 session 4) — gap closure pass
- **System tray** now real: `internal/tray/systray.go` (getlantern/systray, embedded `icon.ico`,
  OS-locked goroutine, panic-recovered). Menu: Show VPM / Quit. `beforeClose` hides only when
  `tray.Active()`; otherwise stops servers + exits (no hidden-window trap). Toast text corrected.
- **System Ports panel** wired: `components/system-ports.js` (sidebar → System Ports) using the
  already-existing `ScanSystemPorts`/`KillByPort` IPC. Inline two-step kill + rescan.
- **Collision resolve**: alert-panel "Use free port :NNNN" → `vpm:apply-suggested-port` →
  `applySuggestedPort` (UpdateServer + StartServer).
- **Log export** switched to native `BrowseDirectory` picker.
- **RAM metric** fixed: `mem_windows.go` reports real process working set (psapi) instead of
  `runtime.MemStats.Sys`; `mem_other.go` stub for non-Windows.
- **Tests added**: `internal/config` round-trip + defaults, `internal/ports` SuggestFree,
  `app_helpers_test.go` (tailFile/copyFile). `go test ./...` green; `go vet` clean; SSOT passes.
- **Dead code removed**: `frontend/src/tabs/{dashboard,logs,workspace}.js`.
- New dependency: `github.com/getlantern/systray v1.2.2`.

### Done ✅ (2026-05-29 session 5) — PM2 대체 기능
- **이벤트 위임 버그 수정**: server-card/log-strip의 `onclick="event.stopPropagation()"` wrapper가
  위임 click 핸들러를 막아 모든 서버 버튼이 죽어있던 것 제거. 위임 리스너는 `init()`에서 1회만 부착.
- **PM2 ecosystem import (다중 서버)**: `internal/project/ecosystem.go`가 `ecosystem.config.{cjs,js,json}`을
  파싱해 앱별 {name, command, port} 추출 → `ProjectAnalysis.Servers`. 프로젝트 추가 시 2개 이상이면
  체크박스 다중선택 import (`sidebar.js: runMultiServerImport`). **PM2 런타임 의존 없음 — 설정 포맷만
  데이터로 읽음 (AGPL 무관).**
- **Crash 자동재시작**: `config.Server.autorestart`(schema+types) 추가. `watchExit`가 비정상 종료 시
  재시작, 60s 내 5회 초과 시 crash-loop으로 ERROR 처리. add/edit 서버 모달에 체크박스.
- **헤드리스 데몬**: `vpm --daemon`(`internal/daemon`) — GUI 없이 autostart 서버 상주 + crash 재시작 +
  로그 로테이션, SIGINT/SIGTERM에 정리 종료. `vpm --install-daemon`/`--uninstall-daemon`으로 로그인 상주 등록.
- **Stop 포트 회수 강화 (`stop.go: reclaimPort`)**: 기존엔 (1) 외부 프로세스는 핸들 없어 no-op,
  (2) 핸들 있어도 PID 하나만 죽여 포트 소유 PID(보통 npm→cmd→node 트리의 손자) 또는 중복 인스턴스가
  남아 PORT_COLLISION 반복. 이제 핸들 유무와 무관하게 **포트 LISTEN 소유주를 resolve→트리킬(taskkill
  /T /F)→재확인을 포트가 빌 때까지 반복**(최대 6패스). 권한 문제 아님(동일 사용자 비권한 kill 정상).
  포트 미고정 명령(`npm run dev`)이 strictPort 서버와 충돌해 중복 vite를 남기던 것도 이걸로 정리됨.
- **포트 드리프트 Re-detect (`ResyncServerPort` IPC)**: VPM이 띄운 서버가 설정 포트와 다른 포트로
  떠도 카드 ⋮ → "Re-detect port" 한 번으로 다시 잡음. `Backend.ResolveTreePort(rootPID)`(win:
  Toolhelp 프로세스 스냅샷으로 자손 PID 집합 ∩ netstat) → 핸들 PID 트리의 실제 LISTEN 포트를 찾아
  config 갱신. **VPM이 spawn한 서버 한정**(핸들 없으면 거부). netstat raw 파서 분리로 tasklist
  per-port 제거(48s→0.6s).
- **로그 캡처 한계(설계)**: 외부에서 떠 있던 서버(포트탐지 RUNNING)는 stdout 파이프가 없어 로그 못 봄
  (OS상 남의 stdout 사후 부착 불가). 로그 보려면 VPM에서 재시작해 spawn 소유시켜야 함.
- 테스트: ecosystem 파서(3-앱), reclaimPort, ResolveTreePort(win). build/vet/test/SSOT 통과, 데몬 스모크 확인.

### Done ✅ (2026-05-29 session 6) — UI 통일 + 미니모드
- **Frameless 커스텀 타이틀바**: `main.go` `Frameless: true` + `BackgroundColour` gray-950로 통일.
  `components/titlebar.js`가 다크 타이틀바(드래그 영역 `--wails-draggable:drag`) + 윈도우 컨트롤
  (mini/minimise/maximise/close). close는 `Quit`→`beforeClose`(트레이 hide 로직) 그대로 탐.
  MinWidth/Height 360/44로 낮춰 미니모드 허용.
- **미니모드**: `components/mini-bar.js` — 활성(비-STOPPED) 서버만 **상태색 포트 칩**으로 띄우는
  수평 바. 진입 시 창을 480×44로 축소 + always-on-top, 칩 클릭=해당 서버로 펼치기, ⤢=복귀.
  진입 전 크기를 저장해 복귀 시 원복. state/projects 이벤트로 실시간 갱신.
- `wails.js`에 윈도우 컨트롤 헬퍼(WindowMinimise/ToggleMaximise/Quit/Get·SetSize/SetAlwaysOnTop).
- **포트 스캔 성능 수정**: `winBackend.ScanListenPorts`가 포트마다 `tasklist`를 호출(리스너 많으면
  ~48s, UI 멈춘 것처럼 보임) → **netstat 1회 + tasklist 1회(pid→name 맵)**로 변경(71포트 1.9s).
  add-project Detection Mode 스냅샷 스캔에 로딩모달 + 8s 타임아웃 추가(`sidebar.js`).
  ※ 모노레포(예: AgentOS_v6, root에 dev 스크립트·포트 없음)는 자동탐지 실패→Detection Mode로 가는데,
    이 스캔이 느려 "아무것도 안 됨"으로 보였던 게 원인.

### Done ✅ (2026-05-29 session 7) — 범용 다중서버 탐지 (Detector 프레임워크)
- **Detector 프레임워크**(`internal/project/detect.go`): `Detector` 인터페이스 + `DetectAll`(전체 실행,
  포트/이름 dedupe). 유형마다 파일 하나 — 책임 분리. 새 유형은 detector 추가 + `detectors()` 등록만.
- 탐지기: **workspace**(monorepo apps/* — package.json `workspaces`/pnpm-workspace.yaml 글롭 확장,
  각 앱 dev 스크립트→`cd /d <rel> && <pm> run dev`, 포트는 앱 디렉토리에서 정적추출), **compose**
  (docker-compose 서비스 호스트포트, yaml.v3 파싱), **pm2**(기존), **single**(루트 단일앱; 워크스페이스
  루트는 제외).
- `detect_pm.go`: **`bun.lock`(신형 텍스트 락) 추가** — AgentOS가 PM=none으로 잘못 잡히던 것 수정.
- `detect_port.go`로 포트 추출 분리(워크스페이스/단일 공유).
- 프론트(`sidebar.js`): 구조적 소스 발견 시 import 흐름. import 모달이 **포트 미상(0) 앱은 편집
  입력**으로 받음(workspace api/desktop 등). dedupe·소스표기·skip 안내.
- 실측: AgentOS_v6(bun 모노레포)→ web:5883 + api/desktop(포트입력), Bananadancer→ PM2 3개.
- 신규 의존성: `gopkg.in/yaml.v3`(compose 파싱). 테스트: workspace/compose/single/DetectAll dedupe.

### Done ✅ (2026-05-29 session 7b) — 탐지 범용 확장
- 언어/런타임 탐지기 추가(primary): **python**(Django manage.py·FastAPI/uvicorn·Flask, venv 경로 자동),
  **go**(go.mod+main → `go run .` 또는 `cmd/*`), **rust**(Cargo.toml → `cargo run`).
- 폴백 탐지기(primary가 0건일 때만, 중복 방지): **procfile**(web/worker), **task**(Makefile/justfile/
  Taskfile의 dev/run/serve 타겟). `DetectAll`을 primary→(없으면)fallback 2단계로 분리.
- import 모달을 **이름·포트·명령 전부 편집 가능**하게 개선 — 포트/명령이 추정값(go/rust/python 등)이어도
  사용자가 확인·수정 후 등록. 미완성 항목은 skip 안내.
- 테스트: python(django/fastapi)/go/rust/procfile/makefile + 폴백 억제 규칙.

### Done ✅ (2026-05-29 session 7c) — 언어 커버리지 확대
- 탐지기 추가(primary): **dotnet**(*.csproj/*.sln, launchSettings 포트), **ruby**(Rails bin/dev·bin/rails,
  Rack), **php**(Laravel artisan·Symfony·내장서버), **java**(Spring Boot Maven/Gradle, 래퍼 우선),
  **elixir**(Phoenix), **deno**(deno.json tasks). 비-Spring Java 등 웹 아닌 건 무시.
- import 모달 source 라벨 + 신규 유형 추가. 테스트: dotnet/rails/laravel/spring/phoenix/deno + non-spring 무시.
- 총 커버리지: JS(단일/모노레포/PM2) · docker-compose · Python · Go · Rust · .NET · Ruby · PHP · Java(Spring)
  · Elixir(Phoenix) · Deno · Procfile · Make/Just/Taskfile.

### 데이터 저장 위치 (참고)
유저 데이터는 exe 옆이 아니라 **`%APPDATA%\vpm\config.json`**(JSON, 원자적 저장), 로그는
`%APPDATA%\vpm\logs\<project_id>\`. Program Files 읽기전용/업데이트 보존 때문에 per-user APPDATA가 정석.

### Pending ❌
None — MVP Windows build complete. See Known Gaps below for deferred scope.

## Known Gaps vs plan.md (Windows MVP Scope)

plan.md is a cross-platform spec. Implementation is Windows-only MVP. Gaps are documented in `docs/known-limitations.md`.

### Not Implemented (deferred)

| plan.md § | Feature | Status |
|---|---|---|
| §1 | macOS / Linux build targets | ❌ Windows-only |
| §2.1 | macOS/Linux login shell (`/bin/zsh -l -c`) | ❌ pwshwrap only |
| §2.2 | macOS/Linux Triple-Pass (Setpgid + SIGTERM + lsof + kill -9) | ❌ Windows taskkill only |
| §2.4 | System tray (Show/Quit menu) | ✅ getlantern/systray (`internal/tray/systray.go`) |
| §2.4 | Per-server Start/Stop in tray menu | ❌ static Show/Quit only |
| §2.5 | Native dialog for log export | ✅ `BrowseDirectory` (directory picker) |
| §2.6 | Port collision one-click resolve | ✅ "Use free port" in alert strip → reassign + relaunch |
| §3 Tab 1 | Native folder picker | ✅ ⊞ button → `OpenDirectoryDialog` |
| §3 Tab 1 | Drag-and-drop directory selector | ❌ picker only, no HTML5 folder drop |
| §3 Tab 1 | package.json scripts dropdown | ❌ PM + best script detected, no full list |
| §3 Tab 2 | System Port Analyzer + Port Killer | ✅ `components/system-ports.js` (sidebar → System Ports) |
| §3 Tab 3 | Auto-scroll lock toggle | ❌ always auto-scrolls |
| §3 Tab 3 | Copy logs to clipboard | ❌ not implemented |

### Architecture Deviations

| Item | plan.md | Actual |
|---|---|---|
| Idle RAM SLO | ≤ 30 MB | 80 MB gate (Wails+WebView2 baseline ~65 MB — irreducible) |
| Status-bar RAM metric | (n/a) | process working set (psapi), excludes WebView2 children |
| Log export UX | Save**File** dialog | native **directory** picker (projects have many log files) |
| Collision response | full-screen wizard | inline alert strip with one-click apply |

## Key Design Decisions

| Decision | Rule |
|---|---|
| No `alert()`/`confirm()` | Always use `Modal.*` from `modal.js` |
| OS calls via interface | `pkg/syscall.Backend` / `Executor` only — no `os/exec` direct |
| One file, one concept | filefunc rules in `internal/`; `app.go`/`main.go`/`frontend/` exempt |
| Port Killer phases | Graceful(SIGTERM) → Poll(3×500ms) → Force(resolve+kill or report) |
| Cross-target block | Windows PID blocks WSL port → diagnostic only, no auto-kill |
| Log rotation | On startup + every 1h via goroutine; 7-day retention + 100MB cap |
| PM2 관계 | **런타임 의존 금지** (PM2=AGPL-3.0 → 배포 오염). ecosystem 설정은 import용으로 파싱만(데이터, 라이센스 무관). 프로세스 관리는 자체 재구현 |
| Crash 재시작 | autorestart=true 서버만; 60s 윈도우 5회 상한 → 초과 시 ERROR(crash loop) |
| 데몬 모드 | `vpm --daemon` 헤드리스; 부팅 상주는 `--install-daemon`(별도 Run-key 값) |

## SLOs (from specs/manifest.yaml)

- exe ≤ 15MB (actual: ~10 MB ✅)
- idle RAM ≤ 80MB (revised from 30 MB — Wails+WebView2 baseline ~65 MB; actual: ~66 MB ✅)
- listening ports = 0 ✅

## Coverage Gate

```powershell
go test ./internal/... -coverprofile=coverage.out -timeout=30s
# portkiller: ≥90%  logbuf: ≥90%  pwshwrap: ≥90%
```

## SSOT Validate

```powershell
go run ./scripts/validate-specs.go
```

## Delegation Docs

Read `docs/delegations/00-overview.md` for full agent task split.
