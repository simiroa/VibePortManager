# Triple-Pass가 nested children (vite의 esbuild worker)을 다 죽였는지 외부 검증.
# Vite는 graceful SIGTERM에 esbuild 워커가 즉시 안 죽는 경향 있음 → Phase 3 force kill로 보장돼야 함.
# 전제: Go 테스트가 StartServer → RUNNING → StopServer.
# 추가 검증은 Go 측에서:
#   - manifest.yaml port_killer.total_grace_ms=1500 이내 STOPPED 도달
#   - tasklist /v 로 node.exe / esbuild 자식 자취 없음
# Hurl은 포트 즉시 재바인딩 가능성으로 사후 증거 제공:

GET http://{{host}}:{{port}}/
# 모든 자식 사망 → connection refused. 200이 오면 좀비 살아있음.
HTTP 200
[Asserts]
duration < 100
