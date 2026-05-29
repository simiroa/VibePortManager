# 40. Wails App: 도메인 + Config + Tray + Autostart

## 목표

Wails v2 app 스켈레톤. Project/Server 도메인. config.json I/O. 시스템 트레이. Autostart 등록.
`app.go`, `main.go`, `internal/project/`, `internal/server/`, `internal/config/`, `internal/tray/`, `internal/logbuf/`.

의존: 10, 20, 30 위임 모두.

## 산출물

### `main.go`

표준 Wails v2 부트스트랩. `app.go`의 `App` struct를 Bind. 단일 윈도우, HideOnClose 동작은 Wails OnClose 훅에서 분기.

### `app.go` (filefunc 예외)

`specs/ipc.yaml`의 모든 메서드를 1:1 노출. 메서드는 얇은 위임 — 실제 로직은 `internal/*`에.

```go
type App struct {
    projects *project.Store
    servers  *server.Manager
    tray     *tray.Manager
    backends map[string]syscall.Backend
}
// 메서드 시그니처는 specs/ipc.yaml과 정확 일치 (validate-specs 강제)
```

### `internal/config/store.go`

```go
//ff:what %APPDATA%/vpm/config.json 직렬화/역직렬화 + 스키마 검증
//ff:why 유저 데이터의 단일 진입점. atomic write로 손상 방지
type Store interface {
    Load() (Config, error)
    Save(Config) error  // tmp 파일 → rename atomic
}
// Config 타입은 scripts/gen-types.go가 config.schema.json에서 생성
```

### `internal/project/`

- `add.go`: 디렉토리 경로 → execution_target 자동 판별 (UNC `\\wsl$\<distro>\` 인식) + 패키지 매니저 감지 (manifest.yaml의 lock 우선순위)
- `detect_pm.go`: lock file 스캔 → PM 결정. 단일 파일이라 단순 fn.
- `remove.go`: 종속 Server 모두 stop → Project 제거

### `internal/server/`

- `manager.go`: 모든 Server의 상태 + 라이프사이클 컨트롤. `map[ServerID]*runtimeServer` + 상태별 mutex.
- `start.go`: pre-flight (port check) → Backend.Spawn → 상태 전이 emit.
- `stop.go`: portkiller.TriplePass 호출.
- `restart.go`: stop → start 순차. 도중 상태 강제.
- `state.go`: state enum (specs/states/server.mmd와 1:1). validate-specs 검증.
- `validate.go`: 등록 시점 검증 — 같은 Project 내 포트 중복 차단.

### `internal/logbuf/buffer.go`

```go
//ff:what Server stdout/stderr → 100ms 배치 → Wails event
//ff:why UI 프리징 방지 (plan.md §2.3)
type Buffer struct { ... }
func (b *Buffer) Write(p []byte) (int, error) // io.Writer 인터페이스
// 내부 goroutine: 100ms ticker → 누적 라인을 events.Emit("server.log.batch")
// 동시에 파일 writer로 분기 (디스크 파일은 즉시 flush, 배치 안 함)
```

### `internal/logbuf/rotation.go`

```go
//ff:what 로그 회전 + retention (CONTEXT.md "Log Retention 정책")
//ff:why 디스크 폭주 방지
func StartCleaner(ctx context.Context, root string)
// 1시간 ticker. 매 tick:
//   1) 파일명 날짜 7일 초과 → 삭제
//   2) Project별 합계가 100MB 초과 → 오래된 것부터 삭제
```

### `internal/tray/`

- `tray.go`: Wails systray API 사용. 메뉴 구성. (라이브러리: Wails 내장 또는 `getlantern/systray`)
- `menu.go`: 동적 메뉴 — Project별 서브메뉴, Server별 [Start]/[Stop] + 상태 색 dot. ListProjects 변경 이벤트 구독.

### `internal/tray/autostart.go`

```go
//ff:what Windows 시작 프로그램 레지스트리 등록/해제
//ff:why §Autostart. golang.org/x/sys/windows/registry 사용
func SetAutostart(enabled bool) error
// HKCU\Software\Microsoft\Windows\CurrentVersion\Run\VPM = "<exe path>" --tray-only
```

### Close 버튼 분기

`main.go`의 Wails OnClose 훅:
```
if 실행 중인 Server 1+ → Hide window + (최초 1회 토스트) → return PreventClose
else → 정상 종료
```

## Server.command의 {PORT} 치환 흐름

```
Start 요청
  → AppPort = Server.port (또는 §2.6 wizard 결정값)
  → shellwrap.Build(cmd, AppPort, env)
  → backend.Spawn(spec) → Handle 보관
  → state.STARTING → (포트 바인딩 확인 후) RUNNING
```

## 동시성

- `server.Manager`는 `sync.Map[ServerID]*runtimeServer`.
- 각 Server는 자기 mutex. 상태 전이는 mutex 안에서만.
- STARTING/STOPPING 중 추가 요청은 즉시 ErrBusy 반환 (UI에서 버튼 disabled로 사전 차단).

## 테스트

- `internal/project/*_test.go`: detect_pm 표 기반.
- `internal/server/*_test.go`: MockExecutor 주입, 모든 상태 전이 경로.
- `internal/logbuf/*_test.go`: 100ms ticker fake clock.
- `app.go`는 통합 테스트(`test/e2e/`)에서만 검증, 단위 테스트 X.

## 완료 정의

- [ ] app.go 메서드가 ipc.yaml과 1:1 일치
- [ ] config.json atomic save/load
- [ ] Server 상태머신이 server.mmd와 1:1
- [ ] Close 버튼 분기 + 최초 1회 토스트
- [ ] Autostart 레지스트리 toggle
- [ ] Tray 동적 메뉴 + Server별 토글
- [ ] go test 통과, 커버리지 ≥ 80%
