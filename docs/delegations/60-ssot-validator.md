# 60. SSOT validator + gen-types

## 목표

yongol 원리의 핵심 가치를 VPM에 이식: **선언 스펙과 코드의 일관성을 기계 검증**.
다른 영역과 병렬 진행 가능.

## 산출물

### `scripts/validate-specs.go`

`go run scripts/validate-specs.go`로 단발 실행. 모든 검증 통과 시 exit 0. 실패 시 첫 위반만 출력 + exit 1.

검증 룰:

1. **manifest.yaml.slo.listening_ports == 0** ↔ `grep -r "net.Listen" .` 결과 0건 (frontend, scripts 제외)
2. **config.schema.json의 모든 필드** ↔ `scripts/gen-types.go` 출력 Go struct의 필드 (이름/타입/required)
3. **ipc.yaml의 methods** ↔ `app.go`에서 발견되는 동일 시그니처 메서드. AST 파싱 (`go/ast`)
4. **specs/states/port_killer.mmd의 state 이름들** ↔ `internal/portkiller/state.go`의 enum 상수 이름들 (1:1)
5. **specs/states/server.mmd의 state 이름들** ↔ `internal/server/state.go`의 enum 상수 이름들
6. **filefunc 룰**: `internal/` 아래 모든 .go 파일에 `//ff:what` 주석 존재 (line 1~10 내). 누락 list.
7. **Hurl 시나리오 매핑**: `specs/scenarios/*.hurl` 파일 각각이 `test/e2e/`의 적어도 한 `*_test.go`에서 참조됨 (단순 grep). 미참조 시나리오 = orphan, 단언 사라진 위험.
8. **Hurl 문법 통과**: 모든 `specs/scenarios/*.hurl`이 `hurl --check` (또는 `hurl run --to-entry 0`)로 파싱 가능.

출력 예:
```
[FAIL] ipc.yaml:42  Method "StartServer(serverID ServerID) error" not found in app.go
[HINT] Add to app.go:
    func (a *App) StartServer(serverID string) error { ... }
```

### `scripts/gen-types.go`

`go run scripts/gen-types.go`로 실행. config.schema.json + ipc.yaml.types → 두 출력:

- `internal/config/types_gen.go` (Go) — `// Code generated. DO NOT EDIT.` 헤더 + 파일 끝에 `// hash: <sha256 of source>` 주석
- `frontend/src/ipc-types.ts` (TS) — 동일

재실행 시:
1. 출력 파일이 존재하면 끝의 hash 주석 읽기
2. 입력 spec의 현재 hash와 비교 → 변경 시만 재생성
3. 사용자가 수동 편집한 흔적(헤더 주석 변경)이 있으면 abort + 경고

### CI 후크 (`.github/workflows/validate.yml` 또는 pre-commit)

```yaml
- run: go run scripts/gen-types.go
- run: git diff --exit-code  # gen 출력 stale 차단
- run: go run scripts/validate-specs.go
- run: go test ./...
```

## 한계 (보고)

- yongol의 코드 재생성 시 user-edit 보존(hash 분할 어노테이션)은 **MVP에서 미구현**. gen-types 출력 파일을 전부 generated 영역으로만 다룸. 부분 편집 가능 영역 없음.
- 룰 1(`net.Listen` grep)은 정적이라 우회 가능 (별칭 import). 완벽 검증은 AST. MVP는 단순 grep으로 시작 + TODO 표시.

## 완료 정의

- [ ] 6개 룰 모두 동작
- [ ] gen-types 산출물이 stale일 때 CI 실패
- [ ] 룰 위반 시 친절한 에러 (위치 + 수정 힌트)
- [ ] pre-commit 또는 GitHub Actions에 통합
