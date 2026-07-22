<!-- 프로젝트의 목적, 구조, 실행 및 테스트 방법을 안내하는 문서입니다. -->

# Auto Stock Trading

토스증권 Open API를 사용하는 Go 기반 주식 자동매매 프로젝트입니다.

현재 기본 실행 모드는 주문을 전송하지 않는 `dry-run`이며, 상대강도·모멘텀·추세·섹터 로테이션·이벤트 점수를 결합하는 전략 엔진을 사용합니다.

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

거래 대상 시장은 `TRADING_MARKET` 환경변수로 선택합니다. 지원 값은 미국 시장의 `us`와 한국 시장의 `kr`이며 기본값은 `us`입니다.

```sh
TRADING_MARKET=us go run ./cmd/trader
TRADING_MARKET=kr go run ./cmd/trader
```

전략 엔진은 시장별 통화 단위의 시가총액과 거래량으로 대상을 선별한 뒤 각 전략의 0~100점 점수를 동일 가중치로 조합합니다. 미국은 시가총액 20억 달러, 한국은 시가총액 2조 원을 기본 하한으로 사용합니다. 종합점수가 85점 이상인 경우에만 매수 후보를 생성합니다. 현재 시장 데이터 연동 전이므로 빈 스냅샷으로 실행하면 주문 신호가 생성되지 않습니다.

## 테스트

```sh
go test ./...
```

실제 인증 정보는 `.env` 또는 실행 환경에만 보관하고 저장소에 커밋하지 않습니다.
