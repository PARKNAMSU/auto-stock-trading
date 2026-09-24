# 작업 인수인계

마지막 갱신: 2026-09-24

## 프로젝트 현황

- Go 기반 토스증권 자동매매 프로젝트다.
- 현재 주문 실행 모드는 `dry-run`만 지원한다.
- 실행 진입점은 `cmd/trader/main.go`다.
- 전략 평가, 기본 위험검사, dry-run 실행 골격이 구현되어 있다.
- 현재 실행은 토스증권 랭킹, 현재가, 종목 정보와 수정 일봉으로 생성한 `MarketSnapshot`을 전략에 전달한다.
- 토스증권 API 인증, 공통 HTTP 클라이언트와 시장 데이터 수집기가 구현되었다.
- MongoDB 연결 기반이 구현되었으며 실행 시 MongoDB 연결/Ping 성공이 필요하다.

## 완료된 작업

- 현재 거래 로직과 구현 공백을 분석했다.
- GitHub 저장소 `PARKNAMSU/auto-stock-trading`에 구현 백로그 이슈 #1~#16을 등록했다.
- GitHub에 `priority:P0`, `priority:P1`, `priority:P2`, `priority:P3` 라벨을 추가했다.
- 토스증권 공식 AI용 문서와 OpenAPI 정본을 루트의 `toss-doc`에 저장하고 세션용 색인을 추가했다.
- 테스트를 외부 테스트 패키지 구조로 이전했다.
  - `test/config` (`config_test`)
  - `test/risk` (`risk_test`)
  - `test/strategy` (`strategy_test`)
- 테스트 이전 작업은 커밋 `8393fbd`에 저장되어 있다.
- 이전 작업 시점에 `go test ./...`가 통과했다.
- 이슈 #2 토스증권 API 인증 및 공통 클라이언트를 구현했다.
  - OAuth 2.0 Client Credentials 토큰 발급과 만료 전 재발급
  - 동시 갱신 직렬화 및 `expired-token`/`invalid-token` 401 재인증
  - 공통 API/OAuth 오류 모델, `Retry-After`, 조회 요청 429/5xx 재시도
  - 외부 호스트로 인증정보 전송 차단 및 로그 내 비밀정보 비노출
  - 범용 JSON 요청 인코딩은 `internal/common`으로 분리
  - `test/tossinvest` 외부 테스트 패키지 추가
- 이슈 #2 구현 후 `go test ./...`, `go test -race ./test/tossinvest`, `go vet ./...`가 통과했다.
- 이슈 #1 시장 데이터 수집기와 `MarketSnapshot` 생성을 구현했다.
  - 시장 거래대금 랭킹 기반 종목 목록 조회
  - 현재가, 발행주식수, 종목 유형, 수정 일봉 수집
  - KRW 원/USD 센트 가격 정규화, 시가총액과 평균 거래량 계산
  - 수정 일봉의 ISO 주봉 집계
  - 수집 시각, 신선도와 결측 필드 기록
  - 공식 API에 없는 섹터를 위한 선택적 `SectorResolver`
  - 실행 진입점에서 빈 스냅샷 대신 실제 수집 결과 전달
- MongoDB 실행 및 애플리케이션 연결 기반 구성을 위한 P0 이슈 #17을 등록했다.
  - 저장 데이터 항목과 컬렉션·스키마 설계는 범위에서 제외했다.
- 이슈 #17 MongoDB 실행 및 연결 기반을 구현했다.
  - Docker Compose 개발 DB(27017, 영속 볼륨)와 테스트 DB(27018, tmpfs) 분리
  - `APP_ENV`별 접속 기본값 및 운영 URI 필수 설정
  - 접속·작업·종료 타임아웃과 최소/최대 커넥션 풀 설정 및 검증
  - 공식 Go 드라이버 v2 기반 시작 Ping, 오류/취소 시 정리, 동시·반복 종료 지원
  - 연결 오류 및 설정 출력의 URI/인증정보 비노출
  - `.env.example`, README 실행/운영/통합 테스트 안내 추가
  - `test/config`, `test/mongodb`에서 잘못된 설정, 접속 거부, handshake 무응답, 취소와 로그 비노출 검증
- 이슈 #17 구현 후 `go test ./...`, `go vet ./...`가 통과했다.
- 로컬 MongoDB 8.0.16에서 `MONGODB_INTEGRATION_URI`를 지정한 `go test -race ./test/mongodb ./test/config -count=1 -v`가 통과했다.
  - 실제 Ping, 취소된 컨텍스트, 동시/반복 Close, 종료 후 Ping 실패를 검증했다.
  - 검증에 사용한 테스트 컨테이너는 중지/제거했다. 개발 볼륨은 변경하지 않았다.

- 이슈 #17 후속 구조 정리로 MongoDB와 토스증권 클라이언트를 `internal/external/mongodb`, `internal/external/tossinvest`로 이동했다.
  - 애플리케이션, 설정, 실주문 골격 및 테스트의 import 경로와 README를 갱신했다.
  - 클라이언트 동작과 공개 API, `test/<package>` 테스트 위치는 유지했다.
  - `gofmt`, `go test ./...`, `git diff --check`를 통과했다. 실제 MongoDB 통합 테스트는 이번 경로 이동에서는 재실행하지 않았다.
- 이슈 #17 후속으로 MongoDB 전용 설정을 `internal/external/mongodb/config.go`로 모았다.
  - `Config`, 비밀값 비노출 포맷, 기본값, 환경변수 로딩과 검증을 MongoDB 패키지가 관리한다.
  - `config.Load()`는 `APP_ENV`를 결정한 뒤 `mongodb.LoadConfig(environment)`를 호출해 전체 설정을 조합한다.
  - MongoDB 설정 테스트는 `test/mongodb/config_test.go`로 이동하고, `test/config`에는 설정 조합과 오류 전달 테스트를 유지했다.
  - `gofmt`, `go test ./...`, `git diff --check`가 통과했다. 실제 DB 통합 테스트는 설정 책임 이동에서 재실행하지 않았다.

## 현재 상태

- 이슈 #17 MongoDB 기반 구현은 커밋 `3b6126c`로 저장되어 있다.
- `internal/external` 패키지 분리 및 MongoDB 설정 책임 이동은 작업 트리에 있으며 아직 커밋/푸시하지 않았다.
- MongoDB 저장 데이터/데이터베이스·컬렉션·인덱스·통계 스키마는 설계하지 않았다.
- 운영 서버의 인증/TLS 및 SRV DNS 환경은 별도 배포 검증이 필요하다. SRV URI 해석은 드라이버/시스템 DNS 제한 시간을 따른다.
- 이슈 #2 구현은 커밋 `623e2e8`로 `origin/main`에 반영되었다.
- 이슈 #2 메서드 이해를 돕는 주석 보강은 커밋 `99a6864`로 `origin/main`에 반영되었다.
- `LiveExecutor` 안전 골격은 커밋 `f1335d7`로 반영되었으며, 실제 주문 API는 아직 호출하지 않는다.
- GitHub 구현 이슈는 우선순위별로 다음과 같이 구성되어 있다.
  - P0: #1~#4, #17 — API, 시장 데이터, 계좌/포지션, 전략 점수, MongoDB 기반 구성
  - P1: #5~#8 — 위험관리, 주문 생명주기, 포지션 크기, 매도 전략
  - P2: #9~#12 — 페이퍼 트레이딩, 스케줄러, 백테스트, 저장소
  - P3: #13~#16 — 실거래 보호장치, 테스트 확대, 설정, 관측성

## 권장 다음 작업

1. GitHub의 열린 `priority:P0` 이슈와 선행관계를 확인한다.
2. 이슈 #17 후속 외부 패키지 분리 및 MongoDB 설정 책임 이동을 검토하고, 커밋/푸시는 사용자 요청 시 진행한다.
3. 수집된 가격 이력에 실제 전략 점수를 계산하는 이슈 #4를 진행한다.
4. 공식 API에 없는 섹터 데이터를 제공할 데이터 소스를 결정한다.
5. 각 작업 후 테스트를 추가하고 `go test ./...`를 실행한다.

## 세션 재개 방법

새 세션에서는 다음 순서로 현재 상태를 확인한다.

```sh
cat HANDOFF.md
git status --short
git log --oneline -5
gh issue list --repo PARKNAMSU/auto-stock-trading --state open
go test ./...
```

사용자에게 진행할 이슈가 지정되지 않았다면 임의로 구현을 시작하기 전에 우선순위와 선행관계를 확인한다.
