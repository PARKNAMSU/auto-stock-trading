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
- `internal/external/tossinvest`: 토스증권 Open API 어댑터
- `internal/external/mongodb`: MongoDB 접속, Ping 및 종료 처리
- `internal/external/sector`: 섹터 분류 공급자 어댑터 자리
- `internal/marketdata`: 시장 데이터 수집 흐름과 스냅샷 생성

외부 시스템 접속 구현은 `internal/external` 아래의 서비스별 패키지로 구분합니다. `domain`, `strategy`, `risk`는 외부 패키지를 직접 참조하지 않습니다. 테스트는 `test/<package>` 위치와 `<package>_test` 패키지명을 유지합니다.

MongoDB 전용 설정 타입, 기본값, `MONGODB_*` 환경변수 로딩과 검증은 `internal/external/mongodb/config.go`에서 관리합니다. 애플리케이션의 `config` 패키지는 `APP_ENV`를 결정한 뒤 `mongodb.LoadConfig(environment)`를 호출해 전체 설정을 조합합니다.

## 실행

먼저 아래 MongoDB 개발 환경을 시작합니다. 애플리케이션은 시작 시 MongoDB 연결과 primary Ping을 확인하고, 실패하면 거래 사이클을 실행하지 않습니다.

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

기본 실행에는 `internal/external/sector.PlaceholderResolver`를 연결해 두었습니다. 실제 분류 공급자가 정해지지 않아 임의 섹터를 만들지 않으며, 스냅샷에 `sector` 결측 경고가 계속 표시됩니다. 공급자와 미국·한국 간 분류 체계 연결은 [이슈 #25](https://github.com/PARKNAMSU/auto-stock-trading/issues/25)에서 후속 구현합니다.

## 테스트

```sh
go test ./...
```

실제 인증 정보는 `.env` 또는 실행 환경에만 보관하고 저장소에 커밋하지 않습니다.

토스증권 API 연결에는 `TOSSINVEST_CLIENT_ID`, `TOSSINVEST_CLIENT_SECRET`을 사용합니다. 계좌 API를 호출할 때는 `GET /api/v1/accounts`가 반환한 `accountSeq`를 `TOSSINVEST_ACCOUNT`에 지정합니다. `TOSSINVEST_BASE_URL`의 기본값은 공식 실전 API인 `https://openapi.tossinvest.com`이며, 공식 문서에는 별도의 모의투자 API 서버가 정의되어 있지 않습니다. 주문 없는 모의 실행은 `TRADING_MODE=dry-run`으로 분리합니다.

공통 클라이언트는 토큰 만료 전 재발급과 401 응답 시 재인증을 수행합니다. 조회 요청의 429 및 일시적인 서버 오류는 `Retry-After`와 지수 백오프를 적용해 재시도하지만, 중복 주문 위험이 있는 POST 요청은 자동 재시도하지 않습니다. 인증정보, 토큰, 계좌 식별자는 요청 로그에 기록하지 않습니다.

## MongoDB 환경 구성 (#17)

Docker Compose v2 이상과 Docker 실행 환경이 필요합니다. MongoDB 이미지는 `mongo:8.0.16`으로 고정합니다. 개발용 데이터는 Docker 볼륨에 보존하고, 테스트용 데이터는 별도 컨테이너의 메모리 파일시스템에 저장합니다. 두 서비스 모두 인증 없는 로컬 전용 구성으로, 호스트의 `127.0.0.1`에만 포트를 공개합니다. 운영 환경에는 이 Compose 구성을 사용하지 않습니다.

```sh
# 개발 DB 시작 및 상태 확인
docker compose up -d --wait mongodb
docker compose ps

# 환경변수 파일은 자동으로 읽지 않으므로 쉘에서 명시적으로 로드
cp -n .env.example .env
# .env에 필요한 토스증권 설정을 로컬에서 입력한 뒤 실행
set -a
. ./.env
set +a
go run ./cmd/trader

# 개발 DB 중지 (볼륨 보존)
docker compose stop mongodb
```

환경별 기본 설정은 다음과 같습니다. 이미 설정한 `MONGODB_URI`는 `APP_ENV`보다 우선하므로, 테스트에서는 명시적으로 테스트 전용 주소를 사용합니다.

| APP_ENV | 기본 MONGODB_URI | 데이터 수명 |
| --- | --- | --- |
| `development` (기본값) | `mongodb://127.0.0.1:27017` | 개발 볼륨에 보존 |
| `test` | `mongodb://127.0.0.1:27018` | 테스트 컨테이너 중지 시 소멸 |
| `production` | 기본값 없음, 환경변수 필수 | 운영 MongoDB에서 관리 |

운영에서는 배포 환경의 비밀 관리 시스템을 통해 `APP_ENV=production`과 `MONGODB_URI`를 주입합니다. 인증정보, `authSource`, replica set/SRV 및 TLS 옵션은 운영 서버에 맞게 URI에 설정합니다. `mongodb://`와 `mongodb+srv://`를 지원하며 인증과 TLS를 활성화한 운영 서버를 사용합니다. URI나 인증정보를 쉘 명령 이력, 배포 로그, 저장소에 넣지 않습니다.

| 환경변수 | 기본값 | 의미 |
| --- | --- | --- |
| `MONGODB_CONNECT_TIMEOUT` | `5s` | 연결 및 서버 선택 제한 시간, 시작 Ping의 상한 |
| `MONGODB_OPERATION_TIMEOUT` | `5s` | 각 Ping/드라이버 작업 제한 시간 |
| `MONGODB_SHUTDOWN_TIMEOUT` | `5s` | 종료 시 연결 정리 제한 시간 |
| `MONGODB_MIN_POOL_SIZE` | `0` | 서버별 최소 풀 크기 |
| `MONGODB_MAX_POOL_SIZE` | `20` | 서버별 최대 풀 크기 (양수, 최소값 이상) |

타임아웃에는 `500ms`, `5s`처럼 양수인 Go duration을 사용합니다. 위 접속·작업·서버 선택 타임아웃과 풀 설정은 URI의 동일 옵션보다 우선합니다. 시작 Ping에는 접속·작업 제한 시간 중 더 짧은 값이 적용됩니다. SRV URI 해석에 필요한 DNS 조회는 드라이버/시스템 resolver의 제한 시간을 따릅니다.

단일 연결 풀을 재사용하고, 실행 완료·오류·SIGINT/SIGTERM 취소 후에는 실행 컨텍스트와 독립적인 제한 시간으로 연결을 정리합니다. 설정 오류는 해당 항목을 표시하며, 연결 오류는 취소·타임아웃·일반 연결 실패로 구분합니다. 서버 응답, URI, 호스트 및 인증정보가 들어갈 수 있는 원본 오류와 드라이버 진단 로그는 출력하지 않습니다.

MongoDB가 없는 일반 테스트와 실제 연결 통합 테스트를 구분합니다. 통합 테스트는 Ping/취소/Disconnect만 수행하며 컬렉션이나 데이터를 생성하지 않습니다.

```sh
go test ./...

docker compose --profile test up -d --wait mongodb-test
APP_ENV=test MONGODB_URI=mongodb://127.0.0.1:27018 \
  MONGODB_INTEGRATION_URI=mongodb://127.0.0.1:27018 \
  go test -race ./test/mongodb -count=1 -v
docker compose --profile test stop mongodb-test
docker compose --profile test rm -f mongodb-test
```

`MONGODB_INTEGRATION_URI`가 없으면 실제 서버가 필요한 테스트는 건너뜁니다. 단위 테스트는 잘못된 설정, 접속 거부, 응답 없는 서버, 취소와 비밀값 비노출을 검증합니다. 접속 실패 시 Docker 서비스 상태, 환경별 포트, 운영 인증 및 TLS 설정을 확인합니다. 이 작업에는 저장 데이터 항목, 데이터베이스/컬렉션 구조, 인덱스와 통계 스키마 설계가 포함되지 않습니다.

접속 수명주기는 MongoDB 공식 문서의 [연결 풀](https://www.mongodb.com/docs/drivers/go/current/connect/connection-options/connection-pools/)과 [컨텍스트 및 타임아웃](https://www.mongodb.com/docs/drivers/go/current/context/) 동작을 기준으로 구성했습니다.
