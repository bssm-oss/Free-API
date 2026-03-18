# 아키텍처

## 전체 구조

```
freeapi/
├── main.go                          # 진입점
├── cmd/                             # CLI 명령어 (cobra)
│   ├── root.go                      # 루트 명령 + 직접 채팅 모드
│   ├── chat.go                      # chat 명령 (스트리밍, 파이프, 컨텍스트)
│   ├── repl.go                      # 인터랙티브 REPL 모드
│   ├── providers.go                 # providers list/status/test
│   ├── history.go                   # history list/show/delete/clear
│   ├── config.go                    # config set/list/init
│   ├── setup.go                     # 설정 위자드 (CLI 감지 + API 키)
│   ├── scan.go                      # AI CLI 스캔
│   ├── models.go                    # 모델 목록
│   ├── export.go                    # 대화 내보내기
│   └── version.go                   # 버전 정보
├── internal/
│   ├── models/
│   │   └── types.go                 # 공유 타입 (Message, Response, etc.)
│   ├── config/
│   │   ├── config.go                # YAML 설정 + 환경변수
│   │   └── config_test.go
│   ├── provider/
│   │   ├── provider.go              # Provider 인터페이스
│   │   ├── httpclient.go            # 공유 HTTP 클라이언트
│   │   ├── cli_provider.go          # CLI 래핑 (gemini, claude, codex, etc.)
│   │   ├── openai_compat.go         # OpenAI 호환 API (Groq, Cerebras, etc.)
│   │   ├── gemini.go                # Google Gemini API
│   │   ├── cohere.go                # Cohere API
│   │   ├── registry.go              # Provider 등록 + 우선순위
│   │   ├── rotator.go               # 자동 로테이션
│   │   └── rotator_test.go
│   └── context/
│       ├── store.go                 # SQLite 대화 저장
│       ├── store_test.go
│       ├── manager.go               # 컨텍스트 관리
│       └── manager_test.go
├── Makefile
├── README.md
└── docs/                            # 문서
```

## 핵심 흐름

### 1. 요청 처리 흐름

```
사용자 입력
    │
    ▼
cmd/chat.go (또는 cmd/repl.go)
    │
    ├─ config.Load() → YAML + 환경변수에서 설정 로드
    │
    ├─ context.NewStore() → SQLite DB 열기
    │
    ├─ context.NewManager() → 컨텍스트 관리자 생성
    │   └─ GetOrContinue() → 새 대화 or 기존 대화 이어가기
    │   └─ BuildMessages() → system + history + user 메시지 조합
    │
    ├─ provider.NewRegistry() → Provider 목록 생성
    │   ├─ DetectCLIs() → 설치된 CLI 자동 감지
    │   └─ createProvider() → API provider 생성
    │
    ├─ provider.NewRotator() → 로테이터 생성
    │   └─ Chat() 또는 ChatStream()
    │       ├─ provider[0].Chat() → 성공 → 응답 반환
    │       ├─ (429/에러) → provider[1].Chat() → ...
    │       └─ 모두 실패 → 에러
    │
    └─ manager.SaveExchange() → SQLite에 저장 (트랜잭션)
```

### 2. Provider 우선순위

```
CLI providers (API 키 불필요, 먼저 시도)
  1. gemini-cli   → gemini --yolo <prompt>
  2. claude-cli   → claude --dangerously-skip-permissions --print <prompt>
  3. codex-cli    → codex exec --full-auto <prompt>
  4. copilot-cli  → copilot -p <prompt> --allow-all-tools
  5. opencode-cli → opencode run <prompt>

API providers (키 필요, CLI 실패 시 fallback)
  6. gemini       → Gemini API (generativelanguage.googleapis.com)
  7. groq         → Groq API (api.groq.com, OpenAI 호환)
  8. cerebras     → Cerebras API (api.cerebras.ai, OpenAI 호환)
  9. mistral      → Mistral API (api.mistral.ai, OpenAI 호환)
  10. openrouter  → OpenRouter (openrouter.ai, OpenAI 호환)
  11. cohere      → Cohere API (api.cohere.ai)
  12. github      → GitHub Models (models.inference.ai.azure.com, OpenAI 호환)
```

### 3. Provider 인터페이스

```go
type Provider interface {
    Name() string
    Chat(ctx, messages, opts) (*Response, error)
    ChatStream(ctx, messages, opts) (<-chan StreamChunk, error)
    IsAvailable() bool
    RateLimitStatus() RateLimitInfo
    MarkRateLimited(RateLimitInfo)
    DefaultModel() string
}
```

모든 provider (CLI, API)가 이 인터페이스를 구현합니다.

### 4. 로테이션 로직

```go
for _, provider := range providers {
    if !provider.IsAvailable() {
        continue  // 키 없거나 rate limited
    }
    resp, err := provider.Chat(ctx, messages, opts)
    if err != nil {
        if isRateLimitError(err) {
            provider.MarkRateLimited(...)
            continue  // 다음 provider로
        }
        continue  // 기타 에러도 다음으로
    }
    return resp  // 성공
}
```

### 5. 컨텍스트 관리

```
SQLite DB: ~/.local/share/freeapi/conversations.db

conversations 테이블
  id, title, system_prompt, created_at, updated_at

messages 테이블
  id, conversation_id, role, content, provider, model,
  tokens_in, tokens_out, created_at
```

슬라이딩 윈도우 전략으로 컨텍스트 크기를 관리합니다:
- 설정된 `max_context_messages` (기본 50)를 초과하면 오래된 메시지부터 제거
- system prompt은 항상 유지

### 6. HTTP 클라이언트

```
SharedClient  - 일반 요청용 (120초 타임아웃)
StreamClient  - 스트리밍용 (타임아웃 없음, SSE 연결 유지)
```

두 클라이언트 모두 connection pooling, keep-alive 설정이 되어 있습니다.

## 설계 결정

### Pure Go (CGO 없음)
- `modernc.org/sqlite` 사용 (pure Go SQLite)
- 크로스 컴파일 용이 (macOS, Linux, Windows)
- `go install`로 바로 설치 가능

### CLI-first 접근
- API 키 없이도 설치된 AI CLI로 바로 사용 가능
- CLI provider가 API provider보다 우선
- `exec.LookPath`로 자동 감지

### OpenAI-compatible 통합
- Groq, Cerebras, Mistral, OpenRouter, GitHub Models 모두 하나의 구현체
- baseURL만 다르고 나머지는 동일한 API 형식
- 커스텀 헤더 지원 (OpenRouter referer 등)

### 트랜잭션 안전성
- `SaveExchange`: user + assistant 메시지를 하나의 트랜잭션으로
- `DeleteConversation`, `ClearAll`: 원자적 삭제
- 프로세스 크래시 시에도 데이터 일관성 보장
