# freeapi

무료 LLM을 하나의 CLI로. 설치된 AI CLI들을 자동 감지하고, 하나가 실패하면 다음으로 자동 전환합니다.

## 설치

```bash
git clone https://github.com/heodongun/freeapi.git
cd freeapi
make install
```

설치 위치: `~/.local/bin/freeapi` (대부분의 시스템에서 PATH에 포함됨)

> **Go가 없다면**: [go.dev/dl](https://go.dev/dl/) 에서 설치하세요.

설치 후 새 터미널을 열고:

```bash
freeapi version    # 동작 확인
freeapi scan       # 사용 가능한 AI CLI 확인
```

## 작동 방식

freeapi는 두 가지 방식으로 LLM을 호출합니다:

### 1. CLI 모드 (API 키 불필요)

이미 설치된 AI CLI를 자동으로 감지해서 래핑합니다:

| CLI | 감지 명령어 | 내부 호출 방식 |
|-----|-----------|-------------|
| **Gemini CLI** | `gemini` | `gemini --yolo <prompt>` |
| **Claude Code** | `claude` | `claude --dangerously-skip-permissions --print <prompt>` |
| **Codex CLI** | `codex` | `codex exec --full-auto <prompt>` |
| **Copilot CLI** | `copilot` | `copilot -p <prompt> --allow-all-tools` |
| **OpenCode** | `opencode` | `opencode run <prompt>` |

이 중 하나라도 설치되어 있으면 **API 키 없이 바로 사용 가능**합니다.

**AI CLI 설치 방법:**

```bash
# Gemini CLI (Google, 무료)
npm i -g @anthropic-ai/gemini-cli
# 또는
brew install gemini

# Claude Code (Anthropic)
npm i -g @anthropic-ai/claude-code

# Codex CLI (OpenAI)
brew install codex

# Copilot CLI (GitHub)
npm i -g @anthropic-ai/copilot-cli

# OpenCode
curl -fsSL https://opencode.ai/install | bash
```

### 2. API 모드 (API 키 필요, 더 빠름)

직접 API를 호출합니다. 모두 무료 tier가 있고 신용카드 불필요합니다:

| Provider | 모델 | 무료 한도 | 환경변수 |
|----------|------|----------|---------|
| **Gemini** | gemini-2.5-flash | 250 요청/일 | `GEMINI_API_KEY` |
| **Groq** | llama-3.3-70b | 1,000 요청/일 | `GROQ_API_KEY` |
| **Cerebras** | llama-3.3-70b | 14,400 요청/일 | `CEREBRAS_API_KEY` |
| **Mistral** | mistral-small | 500K 토큰/분 | `MISTRAL_API_KEY` |
| **OpenRouter** | deepseek-r1 등 27+ | 50 요청/일 | `OPENROUTER_API_KEY` |
| **Cohere** | command-r-plus | 1,000 요청/월 | `COHERE_API_KEY` |
| **GitHub Models** | gpt-4o | 50 요청/일 | `GITHUB_TOKEN` |

API 키 발급:
```bash
# 가장 쉬운 것 하나만 해도 됨
export GROQ_API_KEY="gsk_..."         # https://console.groq.com/keys
export GEMINI_API_KEY="AIza..."       # https://ai.google.dev
export GITHUB_TOKEN="$(gh auth token)" # GitHub CLI가 있다면

# 또는 대화형 설정
freeapi setup
```

### 로테이션 원리

```
freeapi "질문"
  │
  ├─ gemini-cli 시도 → 성공 → 응답 반환
  │
  ├─ (실패/rate limit) → claude-cli 시도 → 성공 → 응답 반환
  │
  ├─ (실패) → codex-cli 시도 → ...
  │
  ├─ (CLI 모두 실패) → Gemini API 시도 → ...
  │
  ├─ (실패) → Groq API → Cerebras → Mistral → ...
  │
  └─ 모두 실패 → 에러 메시지
```

## 사용법

### 기본 채팅

```bash
freeapi "Go의 장점을 설명해줘"
```

### 대화 이어가기

```bash
freeapi "수도가 어디야?"
freeapi chat -c "인구는?"           # 이전 대화 컨텍스트 유지
freeapi chat -c "면적은?"           # 계속 이어감
```

### 특정 provider 지정

```bash
freeapi chat -p gemini-cli "hello"   # Gemini CLI 강제
freeapi chat -p claude-cli "hello"   # Claude CLI 강제
freeapi chat -p groq "hello"         # Groq API 강제 (키 필요)
```

### 파이프 입력

```bash
cat error.log | freeapi chat "이 에러 분석해줘"
git diff | freeapi chat "이 변경사항 리뷰해줘"
```

### 인터랙티브 REPL

```bash
freeapi                    # REPL 모드 진입

you> 안녕
AI> 안녕하세요!

you> /new                  # 새 대화
you> /last                 # 이전 대화로 전환
you> /history              # 대화 목록
you> /status               # provider 상태
you> /quit                 # 종료
```

### 출력 제어

```bash
freeapi chat --raw "답만 줘" > output.txt   # 메타데이터 없이 순수 출력
freeapi chat --no-stream "질문"              # 스트리밍 없이 한번에
```

## 전체 명령어

```
freeapi "message"              메시지 바로 전송
freeapi chat "message"         채팅 (플래그 사용 가능)
freeapi chat -c "message"      마지막 대화 이어가기
freeapi chat -p <name> "msg"   특정 provider 사용
freeapi chat --raw "msg"       순수 출력만
freeapi                        인터랙티브 REPL

freeapi scan                   설치된 AI CLI 스캔
freeapi setup                  설정 위자드
freeapi providers list         provider 전체 목록
freeapi providers test         provider 연결 테스트
freeapi models                 사용 가능한 모델 목록

freeapi history list           대화 기록
freeapi history show <id>      대화 내용 보기
freeapi history delete <id>    대화 삭제
freeapi history clear          전체 삭제
freeapi export <id>            마크다운으로 내보내기

freeapi config set key value   설정 변경
freeapi config list            현재 설정 보기
freeapi version                버전 정보
```

## 설정 파일

`~/.config/freeapi/config.yaml`에 저장됩니다.

```bash
freeapi config set gemini.api_key "AIza..."
freeapi config set groq.model "llama-3.1-8b-instant"
freeapi config set default_system_prompt "한국어로 답해줘"
```

대화 기록: `~/.local/share/freeapi/conversations.db` (SQLite)

## 빌드

```bash
make build          # 현재 플랫폼용 빌드
make install        # ~/.local/bin에 설치
make test           # 테스트 실행
make vet            # 정적 분석
make cross          # macOS/Linux/Windows 크로스 컴파일
```

## 구조

```
cmd/                     CLI 명령어 (cobra)
internal/
  provider/
    cli_provider.go      CLI 래핑 (gemini, claude, codex, copilot, opencode)
    openai_compat.go     OpenAI 호환 API (Groq, Cerebras, Mistral, OpenRouter, GitHub)
    gemini.go            Google Gemini API
    cohere.go            Cohere API
    rotator.go           자동 로테이션 (실패 시 다음 provider)
    registry.go          provider 등록 + 우선순위
  context/
    store.go             SQLite 대화 저장
    manager.go           컨텍스트 관리 (슬라이딩 윈도우)
  config/                YAML 설정 + 환경변수
  models/                공유 타입
```

## 라이선스

MIT
