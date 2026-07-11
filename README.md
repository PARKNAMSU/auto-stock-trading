# Auto Stock Trading

토스증권 Open API를 사용하는 Go 기반 주식 자동매매 프로젝트입니다.

현재 기본 실행 모드는 주문을 전송하지 않는 `dry-run`이며, 기본 전략은 아무 신호도 생성하지 않는 `Hold`입니다.

## 구조

- `cmd/trader`: 애플리케이션 진입점
- `internal/config`: 환경 변수 설정
- `internal/domain`: 핵심 도메인 모델
- `internal/strategy`: 매매 전략
- `internal/risk`: 주문 전 위험 검사
- `internal/trading`: 전략 실행과 주문 처리 흐름
- `internal/tossinvest`: 토스증권 Open API 어댑터

## 실행

```sh
go run ./cmd/trader
```

## 테스트

```sh
go test ./...
```

실제 인증 정보는 `.env` 또는 실행 환경에만 보관하고 저장소에 커밋하지 않습니다.
