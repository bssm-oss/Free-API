# 설정 가이드

## 설정 파일

위치: `~/.config/freeapi/config.yaml`

```yaml
providers:
    gemini:
        api_key: ""                    # 또는 GEMINI_API_KEY 환경변수
        model: gemini-2.5-flash
        priority: 1
        enabled: true
        base_url: https://generativelanguage.googleapis.com/v1beta
    groq:
        api_key: ""                    # 또는 GROQ_API_KEY
        model: llama-3.3-70b-versatile
        priority: 2
        enabled: true
        base_url: https://api.groq.com/openai/v1
    cerebras:
        api_key: ""                    # 또는 CEREBRAS_API_KEY
        model: llama-3.3-70b
        priority: 3
        enabled: true
        base_url: https://api.cerebras.ai/v1
    mistral:
        api_key: ""                    # 또는 MISTRAL_API_KEY
        model: mistral-small-latest
        priority: 4
        enabled: true
        base_url: https://api.mistral.ai/v1
    openrouter:
        api_key: ""                    # 또는 OPENROUTER_API_KEY
        model: deepseek/deepseek-r1:free
        priority: 5
        enabled: true
        base_url: https://openrouter.ai/api/v1
    cohere:
        api_key: ""                    # 또는 COHERE_API_KEY / CO_API_KEY
        model: command-r-plus
        priority: 6
        enabled: true
        base_url: https://api.cohere.ai/v2
    github:
        api_key: ""                    # 또는 GITHUB_TOKEN
        model: gpt-4o
        priority: 7
        enabled: true
        base_url: https://models.inference.ai.azure.com
    cloudflare:
        api_key: ""                    # 또는 CLOUDFLARE_API_TOKEN / CF_API_TOKEN
        model: '@cf/meta/llama-3.3-70b-instruct-fp8-fast'
        priority: 8
        enabled: true
default_system_prompt: You are a helpful assistant.
max_context_messages: 50
context_strategy: sliding_window
db_path: ""                            # 비우면 ~/.local/share/freeapi/conversations.db
```

## 환경변수

환경변수가 config 파일보다 우선합니다 (config에 키가 비어있을 때만 적용).

| Provider | 환경변수 (우선순위순) |
|----------|---------------------|
| gemini | `GEMINI_API_KEY`, `GOOGLE_API_KEY` |
| groq | `GROQ_API_KEY` |
| cerebras | `CEREBRAS_API_KEY` |
| mistral | `MISTRAL_API_KEY` |
| openrouter | `OPENROUTER_API_KEY` |
| cohere | `COHERE_API_KEY`, `CO_API_KEY` |
| github | `GITHUB_TOKEN` |
| cloudflare | `CLOUDFLARE_API_TOKEN`, `CF_API_TOKEN` |

### 환경변수 설정 예시

```bash
# ~/.zshrc 또는 ~/.bashrc에 추가
export GROQ_API_KEY="gsk_xxxxxxxxxxxxx"
export GEMINI_API_KEY="AIzaxxxxxxxxxxxxxxx"
export GITHUB_TOKEN="$(gh auth token)"    # GitHub CLI 있으면
```

## config 명령어

```bash
# 기본 설정 파일 생성
freeapi config init

# 설정 보기 (API 키는 마스킹됨)
freeapi config list

# API 키 설정
freeapi config set gemini.api_key "AIza..."
freeapi config set groq.api_key "gsk_..."

# 모델 변경
freeapi config set gemini.model "gemini-2.5-pro"
freeapi config set groq.model "llama-3.1-8b-instant"

# provider 비활성화
freeapi config set cohere.enabled false

# 시스템 프롬프트
freeapi config set default_system_prompt "한국어로 답해줘"

# 컨텍스트 전략
freeapi config set context_strategy "sliding_window"
freeapi config set max_context_messages 100
```

## 데이터 저장 위치

| 파일 | 경로 | 설명 |
|------|------|------|
| 설정 | `~/.config/freeapi/config.yaml` | YAML 설정 |
| 대화 DB | `~/.local/share/freeapi/conversations.db` | SQLite |

## 우선순위 규칙

1. **환경변수** > config 파일 (API 키)
2. **CLI provider** > API provider (로테이션 순서)
3. **`--provider` 플래그** > 자동 로테이션 (명시적 지정)
4. **`--model` 플래그** > config의 기본 모델

## Provider 비활성화

특정 provider를 사용하고 싶지 않을 때:

```bash
# config에서 비활성화
freeapi config set codex-cli.enabled false

# 또는 API provider 비활성화
freeapi config set cohere.enabled false
```

CLI provider는 해당 바이너리가 없으면 자동으로 비활성화됩니다.
