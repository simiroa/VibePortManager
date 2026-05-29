# Stop 후 포트가 완전 해제되어야 함 (Triple-Pass 성공의 외부 증거).
# 전제: Go 테스트가 StopServer 호출 + STOPPED 이벤트 대기 후 호출.
# 기대: connection refused (Hurl이 [Asserts]로 표현 못함 → exit code 활용).
# 따라서 이 시나리오는 Hurl 단독 실패가 곧 성공. Go 측에서 hurl 종료코드 != 0 을 검증.

GET http://{{host}}:{{port}}/
# 기대: 어떤 응답도 받지 못해야 함. Hurl이 connection error로 종료.
# 만약 200이 오면 Triple-Pass가 실패한 것 → 이 단언이 fail 처리되어야 함:
HTTP 200
[Asserts]
duration < 100   # 만약 어쩌다 200 응답이 와도 추가 차단
