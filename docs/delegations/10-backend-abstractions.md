# 10. 백엔드 추상 (Executor + Backend)

## 목표

모든 OS 호출(`exec.Command`, 프로세스 정보 조회, 포트 점유 조회)을 **테스트 가능한 인터페이스 뒤로 숨김**.
3개 Execution Target(`windows-native`, `wsl:<distro>`, `linux-native`)에 대해 동일 인터페이스로 동작.

## 산출물

### `pkg/syscall/executor.go`

```go
//ff:what OS 명령 실행/프로세스 정보의 단일 진입점
//ff:why os/exec 직접 호출을 막아 테스트에서 mock 가능하게
package syscall

type Executor interface {
    // 실행. ctx 취소로 강제 종료. PGID 정보를 반환해 PortKiller가 사용.
    Spawn(ctx context.Context, spec SpawnSpec) (Handle, error)
    // 프로세스 트리/그룹 시그널 (Phase 1 graceful).
    SignalTree(handle Handle, sig Signal) error
    // PID 단위 강제 종료 (Phase 3).
    KillPID(pid int) error
    // 포트 점유자 PID 조회. 없으면 (0, nil). cross-target도 가능해야 함.
    ResolvePortOwner(port int) (PortOwner, error)
}

type SpawnSpec struct {
    Cmdline    []string          // 셸 래퍼가 이미 조립한 argv
    Cwd        string
    Env        map[string]string // 머지 후 최종 환경
    Stdout     io.Writer
    Stderr     io.Writer
    NewPGroup  bool              // Linux/WSL에서 setpgid
}

type Handle struct {
    PID  int
    PGID int   // Linux/WSL only
    JobObject uintptr // Windows Job 핸들 (선택)
}

type PortOwner struct {
    PID         int
    Description string  // "node.exe (PID 12345)" 식
    Origin      Origin  // same-target | cross-target | unknown
}

type Origin int
const (
    OriginSameTarget Origin = iota
    OriginCrossTarget
    OriginUnknown
)
```

### `pkg/syscall/backend.go`

```go
//ff:what Execution Target 별 Executor 구현 선택
//ff:why 1 Project = 1 Backend. Project.execution_target으로 라우팅
type Backend interface {
    Executor
    ID() string  // "windows-native" | "wsl:Ubuntu" | "linux-native"
    Healthy() error  // distro가 stopped면 wsl --status로 확인 후 부팅 시도
}

func NewBackend(target string, wslDistro string) (Backend, error)
```

### 구현 3종 (각각 own 파일)

- `pkg/syscall/win_backend.go` — Windows
  - Spawn: `CreateProcess` + Job Object로 자식 트리 통제 (가능하면 `syscall.SysProcAttr.CreationFlags = CREATE_NEW_PROCESS_GROUP`)
  - SignalTree: `taskkill /T /PID <pid>`
  - KillPID: `taskkill /F /PID <pid>`
  - ResolvePortOwner: `netstat -ano -p tcp` 파싱 + `tasklist /FI "PID eq <pid>"`

- `pkg/syscall/wsl_backend.go` — WSL bridge
  - 모든 명령 앞에 `wsl.exe -d <distro> -- ` prefix
  - Spawn 내부 명령은 `bash -ilc "<assembled>"`
  - SignalTree: `kill -TERM -<pgid>` (negative PGID)
  - KillPID: `kill -9 <pid>`
  - ResolvePortOwner: `ss -ltnp 'sport = :<port>'` 또는 `lsof -t -i :<port>`. **단, Windows 호스트 측 점유면 nil 반환 → 상위에서 win_backend도 조회**

- `pkg/syscall/linux_backend.go` — Future
  - 인터페이스 충족만, MVP에선 `panic("not implemented")` 또는 빌드 태그로 제외

### `pkg/syscall/mock.go` (테스트)

`MockExecutor` — 호출 기록 + 사전 주입한 응답 반환. 테이블 테스트에서 사용.

## 제약

- 동시성: 한 Executor 인스턴스는 thread-safe. 내부 `sync.Mutex` 필요시 사용.
- 타임아웃: `Spawn` 호출 자체는 즉시 반환(자식 프로세스 시작). `ctx` 취소 → SignalTree → 강제 KillPID 순서.
- 에러: 모두 wrap된 sentinel error로 (`ErrNotFound`, `ErrPermission`, `ErrCrossTarget` ...).

## 테스트 요구

- `pkg/syscall/*_test.go`: 각 backend의 단위 테스트는 mock으로. 실제 OS 호출은 e2e 디렉토리(`test/e2e/`)로 분리, `-tags=e2e`로만 실행.
- 브랜치 커버리지 ≥ 90% (인터페이스 동작 부분).

## 완료 정의

- [ ] 3개 backend 인터페이스 충족 (linux은 stub 가능)
- [ ] `MockExecutor` 구현 완료
- [ ] WSL `cross-target` 분기 진단 동작 (Win backend로 fallback)
- [ ] `go test ./pkg/syscall/...` 통과, 커버리지 ≥ 90%
