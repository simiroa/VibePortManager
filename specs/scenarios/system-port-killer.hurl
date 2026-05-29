# Tab 2 [System Port Analyzer Panel]의 [Port Killer] 동작.
# 전제: Go 테스트가
#   1) VPM 밖에서 vite 직접 spawn (외부 점유자)
#   2) GET 200 응답 확인 (이 파일의 첫 단계)
#   3) VPM의 KillByPort(port) 호출
#   4) 이 파일을 다시 실행 (두 번째 단계 — 응답 없어야 함)
#
# Hurl 한 파일에 두 단계를 표현하기 위해 분기 — Go 측에서 두 번 호출하거나 hurl options로 multi-entry:

GET http://{{host}}:{{port}}/
HTTP 200
[Asserts]
duration < 3000
