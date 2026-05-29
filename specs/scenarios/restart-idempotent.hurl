# Restart 두 번 빠르게 → 최종 RUNNING + HTTP 응답.
# 전제: Go 테스트가 RestartServer 호출 2회 (각 RUNNING 이벤트 대기).
# Restart 동안 상태는 STOPPING → STARTING → RUNNING. STARTING 중 추가 Restart는 ErrBusy → 두 번째 호출은 의도적 지연 후.

GET http://{{host}}:{{port}}/
HTTP 200
[Asserts]
body contains "<!DOCTYPE html"
duration < 5000
