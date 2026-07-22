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

전략 엔진은 시장별 통화 단위의 시가총액과 거래량으로 대상을 선별한 뒤 각 전략의 0~100점 점수를 동일 가중치로 조합합니다. 미국은 시가총액 20억 달러, 한국은 시가총액 2조 원을 기본 하한으로 사용합니다. 종합점수가 85점 이상인 경우에만 매수 후보를 생성합니다.

실행 시 토스증권 시장 거래대금 랭킹에서 대상 종목을 조회하고, 현재가·종목 정보·수정 일봉을 수집해 전략 엔진에 전달합니다. `TOSSINVEST_CLIENT_ID`와 `TOSSINVEST_CLIENT_SECRET`이 필요합니다. 가격은 KRW는 원, USD는 센트 단위로 정규화되고 `PriceScale`에 각각 1과 100이 기록됩니다. 주봉은 공식 API의 수정 일봉을 ISO 주 단위로 집계합니다.

현재 공식 토스증권 OpenAPI에는 섹터 정보가 없으므로 별도 `SectorResolver`를 설정하지 않으면 스냅샷의 `MissingFields`에 `sector`가 기록됩니다. 현재가 시각이 신선도 기준을 넘거나 필수 데이터가 비어 있으면 경고 로그로 명확히 표시됩니다.

## 테스트

```sh
go test ./...
```

실제 인증 정보는 `.env` 또는 실행 환경에만 보관하고 저장소에 커밋하지 않습니다.

토스증권 API 연결에는 `TOSSINVEST_CLIENT_ID`, `TOSSINVEST_CLIENT_SECRET`을 사용합니다. 계좌 API를 호출할 때는 `GET /api/v1/accounts`가 반환한 `accountSeq`를 `TOSSINVEST_ACCOUNT`에 지정합니다. `TOSSINVEST_BASE_URL`의 기본값은 공식 실전 API인 `https://openapi.tossinvest.com`이며, 공식 문서에는 별도의 모의투자 API 서버가 정의되어 있지 않습니다. 주문 없는 모의 실행은 `TRADING_MODE=dry-run`으로 분리합니다.

공통 클라이언트는 토큰 만료 전 재발급과 401 응답 시 재인증을 수행합니다. 조회 요청의 429 및 일시적인 서버 오류는 `Retry-After`와 지수 백오프를 적용해 재시도하지만, 중복 주문 위험이 있는 POST 요청은 자동 재시도하지 않습니다. 인증정보, 토큰, 계좌 식별자는 요청 로그에 기록하지 않습니다.
