# Hurl e2e 시나리오

VPM 자체엔 HTTP가 없으나, **VPM이 관리하는 dev server의 외부 관측 가능한 HTTP 동작**을 Hurl로 검증.
시나리오 = "VPM의 어느 동작이 끝났을 때 관리 대상 서버는 어떻게 보여야 하는가" 의 박제.

## 실행 모델

```
Go e2e 테스트 (test/e2e/*_test.go)
  ↓ VPM 내부 API 직접 호출 (server.Manager 등) — Wails IPC 우회
  ↓ AddProject / AddServer / StartServer / StopServer / RestartServer / KillByPort ...
  ↓ 동작 완료 대기 (server.state.changed 이벤트 또는 polling)
  ↓ exec hurl run <scenario>.hurl --variable port=<n> --variable host=<h>
  ↓ Hurl 종료 코드 0 = 통과
```

Hurl은 외부 의존(전제: hurl ≥ 4.0, vite 가능한 node 환경, WSL Ubuntu).
CI에서 windows-latest 러너에 `choco install hurl` + WSL feature on.

## 변수 규약

모든 .hurl 파일은 다음 변수 사용:
- `{{host}}` — 기본 `localhost`
- `{{port}}` — 시나리오별 주입
- `{{alt_port}}` — collision reassign 시나리오에서만

## 시나리오 목록

| 파일 | 검증 | 의존 fixture |
|---|---|---|
| `server-up.hurl` | Start 후 dev server HTTP 응답 | vite-app |
| `port-released.hurl` | Stop 후 포트 LISTEN 해제 | vite-app |
| `restart-idempotent.hurl` | Restart 후에도 동일 응답 | vite-app |
| `collision-reassign.hurl` | 충돌 시 alt_port에서 응답 | vite-app + 외부 점유자 |
| `triple-pass-zombie.hurl` | Stop 후 자식 트리(esbuild worker 등) 흔적 없음 | vite-app (HMR이 자식 띄움) |
| `wsl-bridge-up.hurl` | WSL distro 내 server → Windows localhost 응답 | wsl-vite-app |
| `system-port-killer.hurl` | KillByPort 후 LISTEN 해제 | 외부 vite (VPM 미관리) |
