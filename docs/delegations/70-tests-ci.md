# 70. 테스트 + CI

## 목표

tsma 원칙(브랜치 커버리지 가이드 테스트 생성)을 수동 운영 형태로 적용.
Windows 러너 필수. WSL 통합 테스트는 Windows 러너의 WSL feature on.

## 산출물

### `go.mod` / Go 모듈

```
module github.com/<user>/vpm
go 1.21
require (
    github.com/wailsapp/wails/v2 v2.x
    golang.org/x/sys ...
)
```

### 테스트 분류

| 디렉토리 | 의도 | 빌드 태그 | CI 실행 |
|---|---|---|---|
| `internal/*/_test.go`, `pkg/syscall/*_test.go` | 단위 — MockExecutor | (none) | 매 PR |
| `test/integration/` | Wails App + 실제 backend, 단 외부 의존 없음 | `-tags=integration` | 매 PR (Windows 러너) |
| `test/e2e/` | 실제 dev server (Vite/Python http.server 등) 띄우고 Triple-Pass 검증 | `-tags=e2e` | 야간/머지 시 |

### `test/e2e/fixtures/`

- `vite-app/` — 최소 Vite 프로젝트 (package.json, vite.config.js, index.html). vite의 dev script는 `vite --port {PORT}`.
- `python-http/` — 비어있는 디렉토리. 명령은 `python -m http.server {PORT}` (collision 시나리오의 외부 점유자 역할).
- `wsl-vite-app/` — 동일 vite-app, WSL Ubuntu에 미러링 (CI에서 자동 복사).

### Hurl 시나리오 연결 (`specs/scenarios/*.hurl`)

VPM 자체는 HTTP가 없으므로 Hurl을 **관리 대상 dev server의 외부 관측 도구**로 사용.
시나리오 파일은 `specs/scenarios/` (SSOT — 검증 의도 박제)이며, 실행 환경은 `test/e2e/`에서 Go 테스트가 조립.

테스트 패턴 (예: `triple_pass_test.go`):

```go
//go:build e2e
package e2e

func TestTriplePassZombie(t *testing.T) {
    h := newHarness(t)  // VPM 내부 API 직접 사용
    defer h.Cleanup()

    proj := h.AddProject("test/e2e/fixtures/vite-app", "windows-native")
    srv := h.AddServer(proj, "dev", "npm run dev -- --port {PORT}", 3030)

    h.StartAndWaitRunning(srv)
    runHurl(t, "specs/scenarios/server-up.hurl", map[string]string{
        "host": "localhost", "port": "3030",
    })  // 단언: HTTP 200

    h.StopAndWaitStopped(srv)
    requireHurlFail(t, "specs/scenarios/triple-pass-zombie.hurl", map[string]string{
        "host": "localhost", "port": "3030",
    })  // 단언: hurl exit != 0 (connection refused 의도)
}
```

`runHurl` / `requireHurlFail` 헬퍼:
```go
func runHurl(t *testing.T, file string, vars map[string]string) {
    args := []string{"run", file}
    for k, v := range vars { args = append(args, "--variable", k+"="+v) }
    out, err := exec.Command("hurl", args...).CombinedOutput()
    if err != nil { t.Fatalf("hurl %s: %v\n%s", file, err, out) }
}
func requireHurlFail(t *testing.T, file string, vars map[string]string) { ... /* exit != 0 기대 */ }
```

### 시나리오 ↔ Go 테스트 매핑

| 시나리오 | Go 테스트 파일 | 핵심 검증 |
|---|---|---|
| `server-up.hurl` | `start_test.go` | StartServer → HTTP 200 |
| `port-released.hurl` | `stop_test.go` | StopServer → connection refused (hurl 실패가 곧 통과) |
| `restart-idempotent.hurl` | `restart_test.go` | RestartServer x2 → HTTP 200 |
| `collision-reassign.hurl` | `collision_test.go` | 외부 점유자 spawn → VPM Start → wizard accept → alt port HTTP 200 + 원 port도 외부 점유자가 살아있음 |
| `triple-pass-zombie.hurl` | `triple_pass_test.go` | Stop → 1.5s 이내 STOPPED + 포트 해제 |
| `wsl-bridge-up.hurl` | `wsl_test.go` | WSL Project Start → Windows localhost HTTP 200 |
| `system-port-killer.hurl` | `port_killer_panel_test.go` | 외부 vite spawn → KillByPort → connection refused |

### Hurl CI 의존성

- Windows runner: `choco install hurl` (또는 GitHub Actions `cargo install hurl`)
- WSL 시나리오: runner에 WSL feature on + Ubuntu distro pre-install + vite-app 미러
- node 20 + python 3 사전 설치 (vite + python http.server fixture)

### `.github/workflows/ci.yml` (Windows 러너)

```yaml
on: [push, pull_request]
jobs:
  validate:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.21' }
      - uses: actions/setup-node@v4
        with: { node-version: '20' }
      - run: go run scripts/gen-types.go
      - run: git diff --exit-code
      - run: go run scripts/validate-specs.go
      - run: go test ./...
      - run: go test -tags=integration ./test/integration/...
      # e2e는 별 잡, scheduled
  build:
    needs: validate
    runs-on: windows-latest
    steps:
      - run: wails build -clean -platform windows/amd64
      - name: Check exe size ≤ 15MB
        run: |
          $size = (Get-Item build/bin/vibe-port-manager.exe).Length / 1MB
          if ($size -gt 15) { exit 1 }
```

### 커버리지 게이트

- `internal/portkiller`, `internal/logbuf`, `internal/pwshwrap`, `internal/wslwrap`: 브랜치 ≥ 90%
- 그 외 internal: ≥ 70%
- `pkg/syscall`: ≥ 90% (mock 기반)
- `app.go`: 측정 제외

### tsma 풍 누락 테스트 인덱싱 (선택, MVP 후)

`scripts/missing-tests.go` — `go/ast`로 각 exported fn 인덱싱 + `*_test.go` 대응 fn 매칭 → 누락 리스트 출력.

## SLO 검증 (CI 별 잡, 빌드 산출물 대상)

1. exe 크기: `(Get-Item ...).Length / 1MB` ≤ 15
2. 청취 포트: 빌드된 exe 실행 후 30초 대기, `netstat -ano -p tcp | findstr LISTENING | findstr <vpm_pid>` 결과 0
3. idle RAM: 30초 대기 후 `tasklist /v /fi "imagename eq vibe-port-manager.exe"`의 메모리 사용량 ≤ 30MB

## 완료 정의

- [ ] CI 워크플로우 동작 (validate / unit / integration / build)
- [ ] 커버리지 게이트 강제
- [ ] SLO 3종 검증 잡 동작
- [ ] e2e 잡 nightly schedule
