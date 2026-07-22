# 작업 인수인계

마지막 갱신: 2026-07-22

## 프로젝트 현황

- Go 기반 토스증권 자동매매 프로젝트다.
- 현재 주문 실행 모드는 `dry-run`만 지원한다.
- 실행 진입점은 `cmd/trader/main.go`다.
- 전략 평가, 기본 위험검사, dry-run 실행 골격이 구현되어 있다.
- 현재 실행은 시장 값만 있는 빈 `MarketSnapshot`을 전달하므로 실제 매매 신호를 만들지 않는다.
- 토스증권 API 인증, 시장 데이터 수집, 실제 지표 계산 및 실주문은 아직 연결되지 않았다.

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

## 현재 상태

- 이 문서 작성 직전 작업 트리는 깨끗했다.
- GitHub 구현 이슈는 우선순위별로 다음과 같이 구성되어 있다.
  - P0: #1~#4 — API, 시장 데이터, 계좌/포지션, 전략 점수
  - P1: #5~#8 — 위험관리, 주문 생명주기, 포지션 크기, 매도 전략
  - P2: #9~#12 — 페이퍼 트레이딩, 스케줄러, 백테스트, 저장소
  - P3: #13~#16 — 실거래 보호장치, 테스트 확대, 설정, 관측성

## 권장 다음 작업

1. GitHub의 열린 `priority:P0` 이슈와 선행관계를 확인한다.
2. 토스증권 API 인증 및 공통 클라이언트 이슈 #2부터 구현한다.
3. 이후 시장 데이터 수집기 이슈 #1을 진행한다.
4. 각 작업 후 테스트를 추가하고 `go test ./...`를 실행한다.

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
