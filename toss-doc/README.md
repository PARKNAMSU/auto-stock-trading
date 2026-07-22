# 토스증권 Open API 로컬 문서

마지막 동기화: 2026-07-22

이 디렉터리는 토스증권이 LLM과 AI coding agent를 위해 공개한 공식 문서와 OpenAPI 정본을 로컬에서 참조하기 위한 문서 묶음이다.

## 읽는 순서

1. `README.md` — 로컬 문서 사용법과 핵심 주의사항
2. `overview.md` — 인증, 요청 흐름, API 그룹, 오류 및 호출 한도 개요
3. `api-reference.md` — 엔드포인트별 Markdown 색인
4. `openapi.json` — 엔드포인트, 파라미터, 스키마, 예시의 최종 정본
5. `llms.txt` — 토스증권이 제공하는 AI 에이전트용 안내 파일

정확한 필드명, 필수 여부, enum, 오류 응답을 구현할 때는 항상 `openapi.json`을 최종 기준으로 삼는다.

## 공식 출처

- 개발자 문서: https://developers.tossinvest.com/docs
- AI 안내: https://developers.tossinvest.com/llms.txt
- 개요 Markdown: https://openapi.tossinvest.com/openapi-docs/overview.md
- API Markdown: https://openapi.tossinvest.com/openapi-docs/latest/api-reference/README.md
- OpenAPI JSON: https://openapi.tossinvest.com/openapi-docs/latest/openapi.json

## 현재 문서 정보

- OpenAPI 규격: `3.1.0`
- API 문서 버전: `1.2.4`
- Base URL: `https://openapi.tossinvest.com`
- 문서화된 API 경로: 27개

## 구현 시 핵심 규칙

- 모든 API는 OAuth 2.0 Client Credentials Grant로 발급한 access token을 사용한다.
- 토큰 발급 API를 제외한 요청은 `Authorization: Bearer {access_token}` 헤더가 필요하다.
- 계좌, 자산, 주문 API는 `X-Tossinvest-Account` 헤더도 필요하다.
- refresh token은 제공되지 않으므로 만료 시 토큰을 다시 발급한다.
- 새 토큰을 발급하면 기존 토큰은 즉시 무효화되므로 동시 갱신을 제어해야 한다.
- 토큰, client secret, 계좌 식별자는 코드나 로그에 기록하지 않는다.
- API 그룹별 호출 한도와 응답 헤더는 `overview.md`와 `openapi.json`을 확인한다.
- 가격과 수량은 문서 스키마에 맞춰 처리하고 부동소수점 오차를 피한다.

## Codex 세션에서 사용

토스증권 API 관련 구현이나 리뷰를 시작할 때 다음 파일을 먼저 확인한다.

```sh
sed -n '1,240p' toss-doc/README.md
sed -n '1,260p' toss-doc/overview.md
rg 'operationId|요청하려는 경로' toss-doc/api-reference.md toss-doc/openapi.json
```

`toss-doc/`은 프로젝트 루트에 있으므로 Git에 추가하면 다른 컴퓨터나 새 clone에도 공유할 수 있다.

## 문서 갱신

공식 문서 버전이 바뀌었을 때 아래 URL의 파일을 다시 내려받고 이 파일의 동기화 날짜와 버전을 갱신한다.

```sh
curl -fsSL https://developers.tossinvest.com/llms.txt -o toss-doc/llms.txt
curl -fsSL https://openapi.tossinvest.com/openapi-docs/overview.md -o toss-doc/overview.md
curl -fsSL https://openapi.tossinvest.com/openapi-docs/latest/api-reference/README.md -o toss-doc/api-reference.md
curl -fsSL https://openapi.tossinvest.com/openapi-docs/latest/openapi.json -o toss-doc/openapi.json
```
