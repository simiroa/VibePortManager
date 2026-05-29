# Start 후 dev server가 HTTP 응답.
# 전제: Go 테스트가 AddProject(vite-app) → StartServer → RUNNING 이벤트 대기 후 호출.

GET http://{{host}}:{{port}}/
HTTP 200
[Asserts]
header "Content-Type" contains "text/html"
body contains "<!DOCTYPE html"
duration < 5000
