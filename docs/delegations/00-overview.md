# 위임 인덱스

목적: 다른 에이전트가 cold-start로 한 영역을 집어 작업 완료 가능하게 함.

## 모든 에이전트 필독

1. [../../CONTEXT.md](../../CONTEXT.md) — 도메인 용어. Project / Server / Execution Target / Backend / Port Collision 정의.
2. [../../plan.md](../../plan.md) — 사용자 기능 명세 원본.
3. [../../specs/manifest.yaml](../../specs/manifest.yaml) — 성능 SLO, 빌드 타깃, 정책 파라미터.
4. [../../specs/ipc.yaml](../../specs/ipc.yaml) — Wails 바인딩 카탈로그. 백/프론트 모두 이걸 기준으로.
5. [../../specs/scenarios/README.md](../../specs/scenarios/README.md) — Hurl e2e 시나리오 인덱스. 외부 관측 가능한 동작의 박제.

## 영역 분할 (의존 그래프 순서)

| # | 영역 | 위임문서 | 의존 |
|---|---|---|---|
| 10 | 백엔드 추상 (Executor + Backend interface) | [10-backend-abstractions.md](./10-backend-abstractions.md) | — |
| 20 | Triple-Pass Port Killer + 포트 스캔 | [20-portkiller.md](./20-portkiller.md) | 10 |
| 30 | 셸 래퍼 (PowerShell / WSL bash) | [30-shell-wrappers.md](./30-shell-wrappers.md) | 10 |
| 40 | Wails App: 도메인 + Config + Tray + Autostart | [40-wails-app.md](./40-wails-app.md) | 10, 20, 30 |
| 50 | Frontend (3 Tabs + Tailwind) | [50-frontend.md](./50-frontend.md) | 40 (ipc.yaml 동결 후) |
| 60 | SSOT validator + gen-types | [60-ssot-validator.md](./60-ssot-validator.md) | 병렬 가능 |
| 70 | 테스트 + CI | [70-tests-ci.md](./70-tests-ci.md) | 10~50 |
| 80 | 프론트엔드 구현 (3 Tabs, Modal, Tailwind) | [80-frontend-impl.md](./80-frontend-impl.md) | 40, 50 |
| 90 | 빌드 검증 (exe 크기, RAM, 포트 SLO) | [90-build-verify.md](./90-build-verify.md) | 모든 영역 |

## 공통 규약

- **One file, one concept** (filefunc). `internal/` 아래는 강제. `app.go` / `main.go` / `frontend/` 예외 (`.filefuncignore`).
- 모든 Go 파일 상단: `//ff:what <한 줄 의도>` `//ff:why <왜 이렇게>` 주석.
- OS 호출은 반드시 `pkg/syscall.Executor` 인터페이스 거쳐야 함. `os/exec` 직접 호출 금지 (테스트 가능성).
- 빌드 SLO: exe ≤ 15MB, idle RAM ≤ 30MB, 청취 포트 0개. CI 게이트.
- 변경한 코드는 `go test ./internal/... -coverprofile=cov` 통과 + 새 브랜치는 `scripts/validate-specs.go` 통과.

## 작업 흐름 (에이전트별)

1. CONTEXT.md + plan.md 통독
2. 자기 영역 위임문서 통독
3. 의존 영역(테이블 마지막 칸) 진행 상황 확인 — 미완료면 mock으로 진행
4. 구현 → 테스트 → validate-specs 통과
5. PR. Reviewer는 CONTEXT.md 용어 사용 일관성 체크.
