# WSL Project Server를 Start → Windows 호스트 localhost로 응답 (WSL2 자동 포워딩).
# 전제: Go 테스트가
#   1) \\wsl$\Ubuntu\home\runner\wsl-vite-app 을 Project로 등록 (execution_target=wsl:Ubuntu)
#   2) StartServer → RUNNING
# 변수: {{host}}=localhost (Windows 호스트), {{port}}=vite 기본 5173 또는 지정.

GET http://{{host}}:{{port}}/
HTTP 200
[Asserts]
body contains "<!DOCTYPE html"
duration < 8000   # WSL 부팅 cold start 고려
