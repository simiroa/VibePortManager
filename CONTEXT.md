# VPM 도메인 용어집

이 파일은 VPM(Vibe Port Manager)의 도메인 언어 SSOT입니다.
**구현 세부사항은 적지 마세요.** 용어와 관계만 정의합니다.

---

## Project

사용자가 등록하는 **단일 디렉토리**. 한 폴더는 정확히 하나의 Project.
같은 디렉토리를 두 번 등록할 수 없음.

- 식별자: `project_id` (UUID, 디렉토리 경로 변경에도 안정)
- 핵심 속성: 디렉토리 절대경로, 감지된 패키지 매니저(npm/yarn/pnpm/bun), 표시 이름, `execution_target`

## Execution Target

Project에 속한 모든 Server가 실행되는 환경. 세 값 중 하나:

- `windows-native` — Windows 호스트에서 직접 실행 (PowerShell 래퍼)
- `wsl:<distro>` — Windows 호스트의 WSL2 distro 안에서 실행 (`wsl.exe -d <distro> -- bash -ilc`)
- `linux-native` — 네이티브 Linux 빌드일 때 (Future, MVP 후순위)

자동 감지 규칙:
- Project 경로가 `\\wsl$\<distro>\...` 또는 `\\wsl.localhost\<distro>\...`로 시작 → `wsl:<distro>`
- 그 외 → `windows-native`
- 사용자가 Tab 1에서 수동 오버라이드 가능

macOS는 MVP 스코프 밖.

## Backend (Execution Backend)

`Execution Target`마다 1개씩 존재하는 내부 추상화. Server Start/Stop/PortPoll/ForceKill의 OS 호출을 캡슐화.
- `WinBackend` (PowerShell + taskkill + netstat)
- `WSLBackend` (`wsl.exe -d <d> -- <cmd>` 위로 Linux 명령 실행, ss/lsof/kill)
- `LinuxBackend` (직접 Linux 명령, MVP 후순위)

## Server

한 Project 내부에서 실행되는 **(실행 명령 + 포트) 한 쌍**. 한 Project는 N개의 Server를 가질 수 있음.

예: `~/my-app` Project 안에 두 개의 Server —
- Server "dev" = `npm run dev` on port 3000
- Server "storybook" = `npm run storybook` on port 6006

- 식별자: `server_id` (UUID)
- 핵심 속성: 명령 문자열, 등록 포트, 현재 상태
- 상태 (확장): `STOPPED` | `STARTING` | `RUNNING` | `STOPPING` | `PORT_COLLISION` | `ERROR`
  - `STARTING`/`STOPPING` 전이 중엔 Start/Stop/Restart 버튼 모두 비활성, spinner 표시
  - 한 Server는 동시에 한 상태만. 동작 중 후속 요청은 무시(큐잉 X, 취소 X)

## 등록 시점 검증

같은 Project 내 두 Server에 같은 포트 등록 불가 → Tab 1 Save 비활성 + 인라인 에러.
다른 Project 간 동일 포트는 허용 (Start 시 §2.6 Collision 마법사로 처리).

> ⚠️ 이전 플랜 문서의 "Workspace"는 모호한 용어였음. 이 프로젝트에서 "Workspace"는 **사용하지 않음** — 항상 Project 또는 Server로 구분.

## Project ↔ Server 관계

- Project: 1 ─ N :Server
- Project 삭제 시 그에 속한 모든 Server는 자동 삭제 + 실행 중이면 먼저 종료
- 트레이 컨텍스트 메뉴(§2.4)는 **Server 단위로 토글** (Project 단위 토글은 없음 — Project엔 "실행 중" 상태가 없기 때문)
- 대시보드 카드(§3 Tab 2)는 **Server 단위로 1장씩 표시**

## Port Collision

VPM이 **Server를 Start하려는 시점**에 그 Server의 등록 포트가 이미 다른 OS 프로세스(VPM 외부 포함)에 의해 점유된 상태.
주: 외부 시스템 스캐너(§3 Tab 2 [System Port Analyzer Panel])가 발견하는 일반적인 "포트 점유"와는 구별 — Collision은 항상 특정 Server의 의도된 포트에 대해서만 정의됨.

### Collision Origin (충돌 원천 분류)

Collision 발견 시 VPM은 점유 PID가 어느 환경에 있는지 진단:
- **same-target**: 충돌 PID가 해당 Server와 같은 Execution Target 안에 있음 → 자동 Force Kill 후보
- **cross-target**: 충돌 PID가 다른 Execution Target에 있음 (예: WSL Server인데 Windows 호스트 측 프로세스가 점유) → **Force Kill 버튼 비활성**, Smart Port Auto-Reassign(§2.6)만 제안
- **unknown**: 진단 실패 (권한 부족 등) → Force Kill 버튼 비활성, 정보 표시만

진단 절차: 충돌 시 양쪽 모두 스캔
- Windows: `netstat -ano`
- 각 실행 중인 WSL distro: `wsl -d <d> -- ss -ltnp` (또는 `lsof -i :<port>`)

## WSL Networking Mode

WSL2의 네트워킹 모드. `wsl --status` 출력으로 자동 감지하여 UI에 표시:
- **mirrored** (Windows 11 22H2+): Windows 호스트와 네트워크 네임스페이스 공유. 한 포트는 Windows + 모든 distro 통틀어 하나만 바인딩 가능.
- **NAT** (기본): distro별 격리. 같은 포트를 distro마다 바인딩 가능, Windows 호스트로는 localhost 포워딩.

이 모드는 Cross-target 진단 결과 해석에 영향 (mirrored면 distro 간 충돌도 가능).

## Project 등록 경로

두 가지 입력 방식 모두 지원:
1. 디렉토리 드래그/선택 — Windows 경로 또는 UNC `\\wsl$\<distro>\...` 인식
2. "WSL Project 추가" 별도 버튼 — distro 드롭다운(현재 설치된 distro 목록) + 그 안의 Linux 경로(`/home/me/app`) 입력

## Process Group / Process Tree

한 Server를 Start하면 OS 레벨에서 프로세스 트리가 생성됨 (예: `npm run dev` → node → vite → esbuild workers).
- Windows: Job Object 또는 PID 트리 (taskkill /T로 통제)
- macOS/Linux: setpgid로 명시 생성한 Process Group

§2.2 Triple-Pass Killer의 종료 대상은 **Server의 Process Group 전체**.

## Port Placeholder

Server의 명령 문자열 안에 쓰이는 리터럴 토큰 `{PORT}`.
Start 시점에 VPM이 실제 사용할 포트 번호로 치환.

예: `vite --port {PORT}`, `npm run dev -- --port {PORT}`

플레이스홀더가 없으면 VPM은 `PORT=<n>` 환경변수만 주입 (Fallback).
UI는 등록 시점에 플레이스홀더 부재를 감지하면 경고를 표시 — Fallback이 동작 안 할 가능성이 있음을 알림.

## Port Reassignment

§2.6 Smart Port Auto-Reassign으로 결정된 추천 포트는 **현재 Server 인스턴스의 런타임에만 적용**.
config.json의 등록 포트는 변경되지 않음. 다음 Start 시 다시 등록 포트로 시도.
영구 변경은 사용자가 Tab 1에서 명시 편집할 때만 발생.

## Log Stream

한 Server **실행 인스턴스**(Start로 시작되어 Stop/Restart로 끝나는 한 번의 실행) 동안 생성되는 **stdout + stderr 통합 스트림**.
두 가지 소비처로 동시 분기:
- UI 라이브 뷰 (§2.3, 100ms 배치)
- 디스크 파일 (§2.5)

## Log File 규칙

- 경로: `%APPDATA%/vpm/logs/<project_id>/<server_id>/<YYYY-MM-DD>_<HHMMSS>_<short-uuid>.log`
- 한 Server 실행 인스턴스 = 한 로그 파일 (Start 시 새 파일 생성, Stop/Restart까지 append). 자정에 날짜가 바뀌어도 같은 파일 유지.
- 파일명의 날짜/시각은 **UTC**. UI 표시 시 로컬 타임존으로 변환.

## Window Close 동작

메인 윈도우 X 버튼 클릭 시:
- **실행 중인 Server가 1개 이상**: 윈도우 숨김(트레이로 이동) — 백그라운드 유지
- **실행 중인 Server가 0개**: 앱 완전 종료

**최초 1회 한정** 안내 토스트: "VPM은 실행 중인 Server가 있을 때 백그라운드로 숨겨집니다. 완전히 종료하려면 트레이 아이콘 → Quit." (다시 보지 않기 체크박스 포함, 설정에 영구 저장)

## Autostart (OS 부팅 시 자동 시작)

- VPM 앱 자체의 Autostart: **기본 OFF**. Settings에서 토글. ON 시 Windows 시작 프로그램 레지스트리(`HKCU\Software\Microsoft\Windows\CurrentVersion\Run`)에 등록.
- Server의 "OS 부팅 시 자동 Start" 동작은 **Server별 `autostart` 플래그**로 제어. Tab 1 등록 폼에 체크박스. 기본 OFF. 플래그가 ON인 Server만 VPM 부팅 시 자동 Start 시도.
- VPM Autostart가 OFF면 Server autostart 플래그도 무력 (VPM 자체가 안 떴으므로).

## Detection Mode

프로젝트 등록 시 정적 분석이 실패했을 때 진입하는 **런타임 포트 감지 상태**.
VPM이 진입 직전 현재 리스닝 포트 스냅샷을 찍고, 사용자가 직접 서버를 실행한 뒤 새로 나타난 포트를 후보로 제시. 사용자 확인으로 종료.

## Port Candidate

Detection Mode에서 발견된 **신규 리스닝 포트 한 개**. 스냅샷 이후 새로 나타난 포트가 후보가 됨. 사용자가 후보 중 하나를 선택 후 확인하면 Server Proposal로 승격.

## Server Proposal

포트 자동 감지(정적 분석 또는 Detection Mode) 결과로 자동 생성된 **Server 초안**.
포트는 확정값, 명령어는 package.json scripts 매칭 제안(없으면 빈 칸), 이름은 script 이름 기반 제안.
사용자가 편집 후 확인하면 실제 Server로 등록됨. 확인 전까지는 config에 저장되지 않음.

## View Filter

사이드바에서 선택된 컨텍스트. 두 값:
- `null` — 전체 (All). 모든 프로젝트의 Server를 표시. 상단에 검색창 노출.
- `project_id` — 특정 Project. 해당 Project의 Server만 표시.

View Filter는 렌더링 필터일 뿐이며, 별도의 UI 컴포넌트나 코드 분기를 만들지 않음 — 단일 Server List 컴포넌트가 View Filter를 받아 렌더.

## Log Retention 정책

이중 보호:
1. **일자 컷오프**: 파일명 날짜가 UTC 기준 7일 이상 지난 파일 삭제
2. **Project별 총 용량 cap**: 한 Project의 모든 server 로그 합이 **100MB** 초과 시, 가장 오래된 파일부터 삭제하여 cap 이하로 유지

두 정책 모두 백그라운드에서 주기적 실행 (예: 1시간 간격) + 앱 시작 시 1회.
