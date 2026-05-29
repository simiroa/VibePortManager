# Unified filterable Server List replaces tab-based navigation

All 뷰와 프로젝트 상세 뷰는 본질적으로 같은 데이터를 다른 필터로 보는 것이므로,
별도 컴포넌트/탭으로 분리하지 않고 단일 Server List 컴포넌트에 View Filter(null=All, project_id=상세)를 주입하는 방식으로 통합했다.
기존 3탭(Port Dashboard / Projects / Log Monitor)은 중복 코드와 분산된 상태를 만들었고, 사용자에게도 "같은 서버 목록을 왜 두 곳에서 보는가"라는 혼란을 줬다.

## Considered Options

- **3탭 유지** — Port Dashboard(모니터링), Projects(CRUD), Log Monitor(로그) 분리. 역할이 명확하나 서버 목록 코드와 상태가 세 곳에 분산.
- **단일 필터 컴포넌트** (채택) — 하나의 컴포넌트, View Filter로 범위 제어. All 뷰에만 검색창 추가.

## Consequences

- Log Monitor 탭 제거. 로그 패널은 Server List 하단에 통합 (카드 클릭 → 해당 서버 로그 자동 열림, 수동 토글 가능).
- 사이드바 구조: All | 프로젝트 목록 | + Add Project 버튼만. 탭 내비게이션 제거.
- CRUD(Add/Remove Server)는 View Filter 무관하게 항상 표시.
