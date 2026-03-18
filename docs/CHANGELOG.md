# Changelog

## v0.1.0 (2026-03-18)

최초 릴리스.

### 기능

- **CLI Provider 자동 감지**: gemini, claude, codex, copilot, opencode
  - 설치된 AI CLI를 자동으로 찾아서 래핑
  - API 키 없이 바로 사용 가능
  - rate limit/quota 시 자동 다음 provider로 전환

- **API Provider 7종**: Gemini, Groq, Cerebras, Mistral, OpenRouter, Cohere, GitHub Models
  - 모두 무료 tier, 신용카드 불필요
  - 환경변수 또는 config 파일로 키 설정
  - OpenAI 호환 API는 단일 구현체로 처리

- **자동 로테이션**: HTTP 429 감지 → 다음 provider → CLI quota 감지 → 다음
  - Retry-After 헤더 파싱
  - 기본 60초 대기 후 재시도

- **대화 컨텍스트**: SQLite 기반
  - `freeapi chat -c` 로 이어가기
  - 슬라이딩 윈도우 (기본 50 메시지)
  - 트랜잭션으로 원자적 저장

- **스트리밍 출력**: SSE 파싱
  - OpenAI/Gemini 스트리밍 지원
  - CLI provider는 전체 응답 후 출력

- **인터랙티브 REPL**: `freeapi` 로 진입
  - /new, /last, /history, /status, /id, /help, /quit
  - 5분 타임아웃

- **파이프 지원**: `cat file | freeapi chat "분석"`

- **CLI 명령어**:
  - `chat`: 스트리밍, -c 이어가기, -p provider, -m model, -s system, --raw, --timeout
  - `providers list/status/test`: provider 관리
  - `history list/show/delete/clear`: 대화 기록
  - `config set/list/init`: 설정
  - `setup`: 대화형 설정 위자드
  - `scan`: AI CLI 감지
  - `models`: 모델 목록
  - `export`: 마크다운/텍스트 내보내기

- **빌드**:
  - Pure Go, CGO 없음
  - `go install github.com/bssm-oss/Free-API@latest`
  - 크로스 컴파일 (macOS/Linux/Windows)
  - 10MB 바이너리

### 테스트

- 23개 단위 테스트 (rotator, store, manager, config)
- go vet 통과
- 실제 API 검증 (GitHub Models GPT-4o, Gemini CLI)

### 알려진 제한

- Codex CLI: OpenAI 크레딧 필요 (무료 한도 소진 시 에러)
- Copilot CLI: 첫 응답이 느릴 수 있음 (MCP 서버 초기화)
- CLI provider는 스트리밍 미지원 (전체 응답 후 출력)
- Cloudflare Workers AI: 아직 미구현 (account_id 필요)
