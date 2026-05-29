# §2.6 Smart Port Auto-Reassign 외부 증거.
# 전제: Go 테스트가
#   1) 외부 점유자(python -m http.server)를 {{port}}로 띄움
#   2) VPM에 동일 {{port}} Server 등록 + Start → collision.detected 이벤트 발생 + suggestedFreePort={{alt_port}}
#   3) 마법사 응답 자동화 (Go가 wizard accept 시뮬레이션) → Server는 {{alt_port}}로 RUNNING
# 검증: 원 포트와 alt 포트 둘 다 응답 (서로 다른 프로세스)

GET http://{{host}}:{{port}}/
HTTP 200
[Asserts]
header "Server" contains "SimpleHTTP"   # python http.server 시그너처

GET http://{{host}}:{{alt_port}}/
HTTP 200
[Asserts]
body contains "<!DOCTYPE html"           # vite-app 시그너처
header "Server" not contains "SimpleHTTP"
