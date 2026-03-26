# AGENTS.md

이 파일은 `freeapi` 저장소에서 작업하는 사람/에이전트를 위한 프로젝트 전용 작업 지침이다.
설계 의도, 현재 동작, 수정 시 주의점, 검증 방법을 가능한 한 한 곳에 모아 둔다.

## 1. 프로젝트 목적

`freeapi`는 여러 무료 LLM 접근 경로를 하나의 Go CLI와 로컬 HTTP 서버로 묶는 도구다.

핵심 목표:

- 설치된 AI CLI를 자동 감지해서 API 키 없이 바로 쓸 수 있게 한다.
- CLI provider가 실패하거나 rate limit에 걸리면 다음 provider로 자동 회전한다.
- API provider도 fallback 체인에 넣어 전체 성공 확률을 높인다.
- SQLite에 대화 컨텍스트를 저장해 direct chat, REPL, HTTP 서버 모두 멀티턴을 유지한다.
- 로컬 서버 모드에서 `/freeapi/chat` HTTP API와 `/swagger` 문서 UI를 함께 제공한다.
- 실행 로그를 JSONL로 남겨 디버깅을 쉽게 한다.

이 저장소에서 작업할 때는 “사용자가 바로 실행해도 되는 제품형 CLI”라는 관점을 유지한다.

## 2. 현재 런타임 동작

### 2.1 주요 진입점

- `main.go`: 앱 진입점
- `cmd/root.go`: 루트 Cobra 명령, direct chat 판별, help/i18n, 전역 로깅 초기화
- `cmd/chat.go`: `freeapi chat ...` 와 direct chat 실행 경로
- `cmd/repl.go`: 인자 없이 실행한 REPL 경로
- `cmd/server.go`: 로컬 HTTP 서버

### 2.2 사용자 모드

현재 사용자는 크게 4가지 방식으로 `freeapi`를 쓴다.

1. direct chat
- 예: `freeapi "hello"`
- 내부적으로 `chatCmd.RunE`를 사용한다.

2. 명시적 chat 명령
- 예: `freeapi chat --provider gemini "hello"`

3. REPL
- 예: `freeapi`
- 여러 턴을 같은 세션에서 주고받는다.

4. HTTP 서버
- 예: `freeapi server`
- `POST /freeapi/chat`

### 2.3 direct chat과 REPL parity

중요:

- direct chat과 REPL은 “대화 메시지 구성”이 같아야 한다.
- direct chat과 REPL은 “CLI provider에 전달되는 prompt 형식”이 같아야 한다.
- direct chat과 REPL은 “stream 요청 처리”가 같은 helper를 타야 drift가 줄어든다.

현재 이 parity는 `cmd/request_stream.go`의 `executeStreamRequest(...)`로 맞춰져 있다.

관련 파일:

- `cmd/chat.go`
- `cmd/repl.go`
- `cmd/request_stream.go`
- `cmd/e2e_test.go`

만약 direct chat과 REPL 중 한쪽만 수정한다면, 반드시 다른 쪽과 동일 동작인지 확인해야 한다.

## 3. 디렉터리 구조

### 3.1 최상위

- `main.go`: 앱 시작
- `Makefile`: 기본 빌드/설치/검증 명령
- `README.md`: 사용자 대상 소개/빠른 시작
- `docs/`: 상세 문서

### 3.2 `cmd/`

- `root.go`: 루트 명령, direct chat 판별
- `chat.go`: direct chat/`chat` 명령
- `repl.go`: REPL
- `request_stream.go`: direct chat/REPL 공통 stream 실행 helper
- `spinner.go`: 터미널 spinner
- `server.go`: HTTP 서버 + Swagger UI + OpenAPI
- `providers.go`: provider 상태/테스트/목록
- `models.go`: 모델 목록
- `history.go`: 대화 기록 관리
- `export.go`: 대화 export
- `config.go`: 설정 변경/조회
- `scan.go`: 설치된 CLI 감지
- `setup.go`: 초기 설정 도우미
- `version.go`: 버전 출력
- `help_i18n.go`: help 텍스트 현지화

### 3.3 `internal/config/`

- `config.go`: YAML config, env override, 기본값

### 3.4 `internal/context/`

- `store.go`: SQLite 열기, migration, CRUD
- `manager.go`: conversation 생성/이어가기, message build, save exchange

### 3.5 `internal/provider/`

- `provider.go`: 공통 Provider 인터페이스
- `registry.go`: provider 생성/정렬
- `rotator.go`: provider rotation
- `cli_provider.go`: 설치된 외부 AI CLI 래퍼
- `gemini.go`: Gemini API
- `cohere.go`: Cohere API
- `openai_compat.go`: Groq/Cerebras/Mistral/OpenRouter/GitHub 계열 공통 구현
- `cloudflare.go`: Cloudflare Workers AI
- `httpclient.go`: 공유 HTTP client
- `state.go`: CLI cooldown 상태 저장

### 3.6 `internal/logging/`

- `logger.go`: JSONL 실행 로그 기록기

## 4. Provider 우선순위와 회전 규칙

### 4.1 기본 원칙

`internal/provider/registry.go`는 다음 규칙을 따른다.

1. CLI provider를 먼저 붙인다.
2. 그 다음 API provider를 priority 순서로 붙인다.

즉 “CLI 먼저, API 나중”이 기본이다.

### 4.2 현재 기본 CLI 우선순위

`internal/provider/cli_provider.go` 기준:

1. `codex-cli`
2. `gemini-cli`
3. `claude-cli`
4. `copilot-cli`
5. `opencode-cli`

기본값은 config로 override 가능하다.

중요:

- CLI 우선순위를 바꾸면 `cmd/e2e_test.go`와 `internal/provider/cli_provider_test.go`가 함께 바뀌어야 한다.
- README/문서에 노출된 우선순위 설명도 같이 확인해야 한다.

### 4.3 CLI provider 실제 호출 방식

현재 CLI 래퍼는 다음 명령 형태를 사용한다.

- `gemini-cli` -> `gemini --yolo --prompt "<prompt>" --output-format text`
- `claude-cli` -> `claude --dangerously-skip-permissions --print "<prompt>"`
- `codex-cli` -> `codex exec --skip-git-repo-check --ephemeral --full-auto "<prompt>"`
- `copilot-cli` -> `copilot -p "<prompt>" --allow-all-tools`
- `opencode-cli` -> `opencode run "<prompt>"`

이 명령 인자는 `internal/provider/cli_provider.go`의 `KnownCLIs()`가 single source of truth다.

### 4.4 API provider 순서

API provider는 `config.DefaultConfig()`에 있는 `priority` 기준으로 정렬된다.
CLI가 전부 실패하거나 비활성화되었을 때 사용된다.

기본 API provider:

- `gemini`
- `groq`
- `cerebras`
- `mistral`
- `openrouter`
- `cohere`
- `github`
- `cloudflare`

### 4.5 Rotator 규칙

`internal/provider/rotator.go`는:

- `IsAvailable()`가 false면 skip
- rate limit error면 다음 provider로
- 일반 error도 다음 provider로
- 성공한 첫 provider의 응답을 반환

이 로직은 `chat`, `repl`, `server` 모두 공통으로 사용한다.

## 5. 요청/대화 처리 규칙

### 5.1 메시지 구성

대화 payload는 `internal/context/manager.go`의 `BuildMessages(...)`가 담당한다.

구성 순서:

1. system prompt
2. 저장된 conversation history
3. 새 user 입력

`sliding_window` 전략일 때 `max_context_messages`를 넘으면 오래된 history를 잘라낸다.

### 5.2 CLI prompt 구성

CLI provider용 prompt는 `internal/provider/cli_provider.go`의 `extractPrompt(...)`가 만든다.

형식:

- `System instructions:`
- `Conversation:`
- `User:`
- `Assistant:`

이 포맷은 direct chat과 REPL에서 동일해야 한다.

### 5.3 stream 처리

현재 stream path의 공통 진입점은 `cmd/request_stream.go`다.

이 함수는:

- provider override 적용
- spinner 표시
- chunk 수집
- stream interrupted 상황 처리

를 공통으로 맡는다.

만약 REPL과 direct chat에서 응답 출력 방식이 달라져야 한다면, 가능하면 “출력 formatting”만 갈라야 하고 “provider 요청 실행”은 계속 공통 helper를 쓰는 게 좋다.

### 5.4 non-stream 처리

직접 `Chat()`을 호출하는 non-stream 경로는 현재 direct chat/HTTP 서버에서만 쓴다.

REPL은 기본적으로 stream path를 사용한다.

## 6. HTTP 서버

### 6.1 엔드포인트

`cmd/server.go` 기준:

- `GET /` -> `/swagger` redirect
- `GET /swagger` -> Swagger UI
- `GET /openapi.json` -> OpenAPI 3.0 spec
- `GET /healthz` -> `{ "status": "ok" }`
- `POST /freeapi/chat` -> 채팅 요청

### 6.2 서버 UX 원칙

`freeapi server`는 단순히 포트만 띄우면 안 된다.
브라우저로 열었을 때 바로 문서를 볼 수 있어야 한다.

즉, 루트(`/`)는 사람 기준 entrypoint이고, `/healthz`는 기계 기준 entrypoint다.

### 6.3 Swagger/OpenAPI 원칙

현재 구현은:

- Swagger UI HTML은 로컬에서 렌더링
- Swagger JS/CSS는 CDN 사용
- spec은 `openAPISpec()`가 직접 JSON 생성

주의:

- API surface가 작기 때문에 spec generation tooling을 무리하게 도입하지 않는다.
- endpoint/response schema가 바뀌면 `openAPISpec()`도 같이 수정해야 한다.
- `cmd/server_test.go`에 route/spec 테스트를 같이 유지한다.

### 6.4 HTTP 서버 변경 시 체크리스트

- `newServerMux()`에 route 추가
- `serverCmd.Long`의 endpoint 설명 업데이트
- startup stderr 메시지 업데이트 필요 여부 확인
- `README.md`와 `docs/USAGE.md` 업데이트
- `cmd/server_test.go`에 route test 추가

## 7. 실행 로그

### 7.1 로그 파일

기본 로그 경로:

- `~/.local/share/freeapi/logs/freeapi.log`

형식:

- JSONL

설정:

- `log_path`
- `log_level`

환경변수:

- `FREEAPI_LOG_PATH`
- `FREEAPI_LOG_LEVEL`

### 7.2 로그 레벨

- `error`
- `info`
- `debug`

### 7.3 로그 원칙

기본 원칙:

- prompt 원문과 response 원문은 저장하지 않는다.
- 길이, provider, model, elapsed time, error, request metadata 같은 메타데이터만 남긴다.

왜냐하면:

- 디버깅에는 충분한 정보가 남아야 한다.
- 사용자의 민감한 입력이 기본 로그에 남아선 안 된다.

### 7.4 현재 주요 로깅 지점

- `cmd/root.go`: 명령 시작/종료, typo
- `cmd/chat.go`: 채팅 요청 시작/종료/에러
- `cmd/repl.go`: REPL 시작/종료, stream/save 오류
- `cmd/server.go`: HTTP 요청 시작/종료/거절/오류
- `internal/provider/rotator.go`: provider 시도/skip/성공/실패
- `internal/provider/cli_provider.go`: subprocess start/timeout/rate limit/error/success
- `internal/context/store.go`: DB open/migrate/pragma 오류

### 7.5 로그 관련 작업 시 주의

- 디버깅 편의보다 개인정보 최소 수집을 우선한다.
- 로그 필드를 추가할 때 prompt/response raw content를 넣지 않는다.
- 새로운 주요 flow를 만들면 최소 `start`/`finish`/`error` 정도는 남긴다.

## 8. 설정 시스템

### 8.1 설정 원천

설정은 두 곳에서 온다.

1. `~/.config/freeapi/config.yaml`
2. 환경변수

환경변수는 config 파일보다 우선한다.

### 8.2 설정 변경 규칙

`config set`에서 지원하는 key를 추가하거나 바꿀 때는 보통 아래를 같이 봐야 한다.

- `cmd/config.go`
- `internal/config/config.go`
- `cmd/config_test.go`
- `internal/config/config_test.go`
- `docs/CONFIGURATION.md`
- `README.md`

### 8.3 비밀값 처리

env에서 읽은 값이 config 파일에 재저장되면 안 된다.

이 규칙 때문에:

- 읽기용: `config.Load()`
- 파일 수정용: `config.LoadRaw()`

를 구분한다.

이 구분을 깨면 사용자의 env-secret이 config 파일에 남을 수 있다.

## 9. SQLite / 컨텍스트 저장

### 9.1 저장 파일

- `~/.local/share/freeapi/conversations.db`

### 9.2 설계 원칙

- pure Go sqlite 사용
- 단일 바이너리 배포 용이
- CGO 없이 설치 가능

### 9.3 저장 규칙

대화 저장은 `SaveExchange(...)`로 user/assistant를 한 트랜잭션에 저장한다.

즉:

- user만 저장되고 assistant가 안 저장되는 중간 상태를 피하려고 한다.

### 9.4 DB 관련 수정 시 주의

- migration이 필요한지 먼저 판단
- `busy_timeout`, `journal_mode`, connection count 설정을 함부로 제거하지 않는다
- conversation/title/updated_at 갱신 규칙을 깨지 않는다

## 10. 문서 원칙

사용자-facing 동작이 바뀌면 문서도 같이 바꿔야 한다.

최소 확인 대상:

- `README.md`
- `docs/USAGE.md`
- `docs/CONFIGURATION.md`
- `docs/ARCHITECTURE.md` (구조나 우선순위 설명이 바뀌면)

특히 다음은 문서 누락이 자주 생긴다.

- provider 우선순위 변경
- 새 config key 추가
- 서버 endpoint 추가
- 로그 경로/레벨 추가
- REPL/direct chat 동작 변화

## 11. 테스트/검증 규칙

### 11.1 기본 검증

작업 후 기본적으로 돌릴 것:

```bash
go test ./...
go vet ./...
```

필요하면:

```bash
go build ./...
make test
make vet
```

### 11.2 변경 영역별 추가 검증

1. provider/CLI 관련 변경

- `internal/provider/cli_provider_test.go`
- `cmd/e2e_test.go`

2. REPL/direct parity 변경

- `cmd/e2e_test.go`의 direct vs REPL prompt shape 테스트 확인

3. 서버 변경

- `cmd/server_test.go`
- 가능하면 실제 `freeapi server` + `curl` smoke

4. 설정 변경

- `cmd/config_test.go`
- `internal/config/config_test.go`

5. 로깅 변경

- `internal/logging/logger_test.go`
- 실제 로그 파일 생성 smoke가 있으면 더 좋다

### 11.3 테스트 작성 원칙

- CLI 관련 E2E는 fake binary 스크립트를 써서 외부 실제 CLI에 의존하지 않게 유지한다.
- 테스트가 로컬 설치된 `codex`, `gemini`, `claude`를 accidentally 호출하지 않게 `PATH`를 명시적으로 제어한다.
- provider prompt 포맷을 검증할 때는 fake CLI가 기록한 prompt 파일을 읽는 방식이 안정적이다.

## 12. 수정 시 자주 깨지는 부분

### 12.1 REPL/direct drift

한쪽만 고치면 다른 쪽이 다른 prompt나 다른 timeout을 쓸 수 있다.

대응:

- 공통 helper 유지
- E2E parity 테스트 유지

### 12.2 provider 우선순위 문서 mismatch

코드만 바꾸고 README/ARCHITECTURE를 안 바꾸면 설명이 곧바로 틀어진다.

### 12.3 config key 추가 후 문서/테스트 누락

`config set` 동작만 추가하고 `DefaultConfig`, env override, docs, tests를 빼먹기 쉽다.

### 12.4 Swagger spec drift

`serverChatRequest`/`serverChatResponse` 필드가 바뀌었는데 `openAPISpec()`를 안 바꾸면 docs가 거짓이 된다.

### 12.5 raw 로그 유출

prompt/response를 디버깅 편하다고 바로 로그에 넣으면 안 된다.

## 13. 작업 우선순위 원칙

이 저장소에서는 다음 우선순위를 지킨다.

1. 사용자 실행 흐름이 깨지지 않는가
2. direct chat / REPL / server 간 동작이 일관적인가
3. fallback과 timeout이 실용적인가
4. 문서가 현재 동작을 정확히 설명하는가
5. 로그가 디버깅에 충분한가

## 14. 에이전트/기여자 체크리스트

작업 전:

- 관련 코드 경로 확인
- 이미 있는 테스트 확인
- README/문서 영향 여부 판단

작업 중:

- 작은 범위로 수정
- direct chat, REPL, server 중 영향 경로 명확히 파악
- prompt/response raw content는 로그에 넣지 않기

작업 후:

- `go test ./...`
- `go vet ./...`
- 문서 동기화
- 필요 시 실제 smoke test

## 15. 빠른 참조

자주 보는 파일:

- `cmd/chat.go`
- `cmd/repl.go`
- `cmd/request_stream.go`
- `cmd/server.go`
- `cmd/spinner.go`
- `internal/provider/cli_provider.go`
- `internal/provider/registry.go`
- `internal/provider/rotator.go`
- `internal/context/manager.go`
- `internal/context/store.go`
- `internal/config/config.go`
- `internal/logging/logger.go`

자주 쓰는 명령:

```bash
go test ./...
go vet ./...
go build ./...
make install
freeapi scan
freeapi providers list
freeapi server
```

문서:

- `README.md`
- `docs/USAGE.md`
- `docs/CONFIGURATION.md`
- `docs/ARCHITECTURE.md`

이 파일의 목표는 “다음 사람이 이 저장소를 열었을 때, 어디를 어떻게 건드려야 하는지 한 번에 이해하게 만드는 것”이다.
