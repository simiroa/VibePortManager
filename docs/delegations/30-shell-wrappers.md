# 30. 셸 래퍼 (PowerShell / WSL bash)

## 목표

Server.command 문자열을 OS별로 실행 가능한 argv로 변환.
{PORT} 플레이스홀더 치환. 환경변수 머지. PATH 상속(NVM/FNM/asdf/mise 호환).

패키지: `internal/pwshwrap/`, `internal/wslwrap/`
의존: `pkg/syscall.Executor`

## 산출물

### `internal/pwshwrap/wrap.go`

```go
//ff:what Windows 명령을 powershell.exe argv로 조립
//ff:why 사용자의 OS-level PATH/NVM 상속 보장
func Build(cmd string, port int, env map[string]string) syscall.SpawnSpec
// 산출: powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "cmd /c <substituted>"
// env: 호스트 os.Environ() 머지 + 사용자 override + PORT=<port>
```

### `internal/pwshwrap/placeholder.go`

```go
//ff:what {PORT} 토큰 치환 + PORT 환경변수 fallback
//ff:why 프레임워크별 PORT 인식 차이 흡수 (Vite/Storybook은 CLI flag만 받음)
func Substitute(cmd string, port int) (substituted string, placeholderFound bool)
```

플레이스홀더 부재 시 UI에 경고 → Server 등록 시 frontend 측에서 1차 검증, backend에서 한 번 더.

### `internal/wslwrap/wrap.go`

```go
//ff:what WSL distro 안의 bash -ilc 호출 argv 조립
//ff:why .bashrc 기반 NVM/FNM init이 interactive+login에서만 동작
func Build(distro, cmd string, port int, env map[string]string) syscall.SpawnSpec
// 산출: wsl.exe -d <distro> -- bash -ilc "<cmd with {PORT} substituted>"
// env는 WSLENV 메커니즘으로 통과: 'WSLENV=PORT/u' 등
```

### `internal/wslwrap/distros.go`

```go
//ff:what 설치된 WSL distro 목록
//ff:why Tab 1 "WSL Project 추가" 드롭다운, execution_target 자동 감지
func List() ([]Distro, error)
// 구현: `wsl.exe --list --quiet` 파싱, UTF-16 LE 디코딩 주의
type Distro struct {
    Name    string
    Default bool
    State   string  // "Running" | "Stopped"
}
```

### `internal/wslwrap/network_mode.go`

```go
//ff:what WSL2 NAT vs mirrored 감지
//ff:why cross-target 충돌 해석에 필요 (CONTEXT.md "WSL Networking Mode")
func Detect() (Mode, error)
// 구현: `wsl.exe --status` 파싱 또는 ~/.wslconfig 읽기. 실패시 Mode=NAT (기본값)
```

## 핵심 알고리즘 노트

### Interactive + Login 셸 결정

- bash: `-i -l -c "<cmd>"` (또는 `-ilc`)
- zsh (WSL distro에 zsh 사용자): `-i -l -c`
- 사용자 셸 감지: `wsl -d <d> -- getent passwd $USER | cut -d: -f7`
- 기본: bash 강제 (대부분의 dev 도구 안내가 bash 가정)
- 사용자 오버라이드는 config.json `settings.shell_override[<target_id>]`

### Windows 측 환경 머지

PATH 머지 순서:
1. 호스트 `os.Environ()` 캡처 (PowerShell 시작 전)
2. 사용자 정의 env (Server별, future feature)
3. VPM이 주입: `PORT=<port>`

### {PORT} 치환 규칙

- 케이스 민감 (`{PORT}` 만, `{port}` 무시)
- 정규식: `\{PORT\}` → 단순 string replace
- 인용부호 내부도 치환: `"--port {PORT}"` → `"--port 3001"`

## 보안

- PowerShell `-Command "cmd /c <user_input>"` 인용 escaping 필수. 사용자 명령에 `"` 포함 시 `\"`로 이스케이프.
- WSL 마찬가지로 bash 인자 내부 `"`/`$` escaping.
- 단, 이건 권한 상승이 아닌 단순 quoting bug 방지용 (사용자가 자기 명령을 자기 컴퓨터에 실행하는 시나리오라 RCE 위협 모델 아님).

## 테스트

- 표 기반 `placeholder_test.go`: 다양한 명령 입력 → 기대 출력.
- `wrap_test.go`: 최종 argv가 manifest.yaml의 `execution_targets.shell_command`와 일치 검증.
- `distros_test.go`: UTF-16 LE BOM 디코딩 단위 테스트 (실제 `wsl.exe` 출력 fixture).

## 완료 정의

- [ ] Build()가 manifest.yaml의 shell_command 패턴 정확히 사용
- [ ] {PORT} 치환 + PORT env fallback 둘 다 동작
- [ ] WSL distro 목록/state 정확
- [ ] Networking mode 감지 (NAT/mirrored)
- [ ] go test 통과
