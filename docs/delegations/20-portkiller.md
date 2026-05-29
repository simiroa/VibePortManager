# 20. Triple-Pass Port Killer + Port Scan

## 목표

plan.md §2.2의 Triple-Pass 정확 구현. [../../specs/states/port_killer.mmd](../../specs/states/port_killer.mmd) 상태머신과 1:1 일치.
패키지: `internal/portkiller/`, `internal/ports/`.

의존: `pkg/syscall.Backend` ([10-backend-abstractions.md](./10-backend-abstractions.md))

## 산출물

### `internal/portkiller/state.go`

```go
//ff:what port_killer.mmd 의 상태 enum
//ff:why 다이어그램과 코드 1:1 일치 → validate-specs.go 검증
type State int
const (
    GracefulSent State = iota
    Polling
    Released
    ResolveBlocker
    ForceKill
    CrossTargetReport
    UnknownBlocker
)
```

### `internal/portkiller/graceful.go`

```go
//ff:what Phase 1: 프로세스 트리에 graceful signal
//ff:why 자식까지 끄려면 트리/그룹 단위 시그널 필요
func Graceful(b syscall.Backend, h syscall.Handle) error
```

### `internal/portkiller/poll.go`

```go
//ff:what Phase 2: 500ms x 3회 포트 TCP probe
//ff:why 양보 시간 부여. 정상 종료 케이스의 다수가 여기서 끝남
func Poll(ctx context.Context, b syscall.Backend, port int) (released bool, err error)
// 내부 루프: manifest.yaml port_killer.poll_count / poll_interval_ms 사용
```

### `internal/portkiller/force.go`

```go
//ff:what Phase 3: 실제 점유 PID 해결 후 강제 종료
//ff:why graceful로 안 죽는 좀비를 물리적으로 해제
func Force(b syscall.Backend, port int) (Result, error)

type Result struct {
    State   State
    Killed  bool
    Origin  syscall.Origin
    PortOwner syscall.PortOwner
}
```

### `internal/portkiller/triple_pass.go`

전체 파이프라인 orchestrator. `Graceful → Poll → (Released | Force) → Result` 시퀀스. UI에 통보할 collision 이벤트 페이로드 조립.

### `internal/ports/scan.go`

```go
//ff:what 시스템 전체 LISTEN 포트 + 점유 PID 스냅샷
//ff:why Tab 2 [System Port Analyzer Panel] 데이터 소스
//
// 모든 활성 Backend(Windows + 실행 중 WSL distro들)를 합쳐 통합 목록 반환
func ScanAll(backends []syscall.Backend) ([]PortEntry, error)

type PortEntry struct {
    Port int
    PID  int
    ProcessName string
    BackendID string  // "windows-native" | "wsl:Ubuntu"
}
```

### `internal/ports/suggest.go`

```go
//ff:what §2.6 Smart Port Auto-Reassign — 가장 가까운 빈 포트 찾기
func SuggestFree(b syscall.Backend, startFrom int) (int, error)
// 알고리즘: startFrom+1 부터 +50 까지 incremental probe. 못 찾으면 random ephemeral 추천.
```

## 핵심 알고리즘 노트

### Force 단계의 cross-target 분기

WSL Server를 죽이려는데 포트 점유자가 Windows 호스트에 있을 수 있음.
순서:
1. 본인 Backend의 `ResolvePortOwner(port)` 호출
2. nil이면 다른 모든 Backend 순회해 `ResolvePortOwner` 시도
3. 본인 Backend에서 찾힘 → `OriginSameTarget` → 강제 종료 실행
4. 다른 Backend에서 찾힘 → `OriginCrossTarget` → **종료하지 않고** Result만 반환 (UI에 안내 + Smart Reassign 제안)
5. 어디서도 못 찾음 → `OriginUnknown` → 모달

### Port Polling 정확성

TCP probe 방식 두 가지:
- (a) `net.Listen("tcp", ":<port>")` 시도 → 즉시 close. 가장 정확하나 권한 충돌 가능
- (b) `Backend.ResolvePortOwner(port)` 호출 결과 nil 여부

**권장: (a) 우선, 실패 시 (b)**. (a)는 LISTEN 상태가 아닌 TIME_WAIT까지 잡지 못하므로 SO_REUSEADDR 한계 고려.

## 테스트

- `internal/portkiller/*_test.go`: MockExecutor 주입. 모든 상태 전이 경로 커버.
- 표 기반: `(input port state, mock responses) → expected (final State, Result, events)`.
- 커버리지 ≥ 90%.

## 완료 정의

- [ ] state.go의 enum과 port_killer.mmd 상태가 1:1 일치 (`scripts/validate-specs.go` 통과)
- [ ] cross-target 분기 정확 동작 (테이블 테스트로 입증)
- [ ] `ScanAll`이 모든 활성 backend 통합 결과 반환
- [ ] `SuggestFree`가 50개 시도 내 빈 포트 찾기
- [ ] go test 통과
