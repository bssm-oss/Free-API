# Provider 가이드

## CLI Provider

설치된 AI CLI 도구를 자동으로 감지해서 래핑합니다. API 키가 필요 없습니다.

### Gemini CLI

```bash
# 설치
npm install -g @google/gemini-cli
# 또는
brew install gemini-cli

# freeapi 내부 호출
gemini --yolo <prompt>
```

- Google Gemini 모델 사용
- 무료, Google 계정만 필요
- `gemini` 명령어 첫 실행 시 인증 안내

### Claude Code

```bash
# 설치
npm i -g @anthropic-ai/claude-code

# freeapi 내부 호출
claude --dangerously-skip-permissions --print <prompt>
```

- Anthropic Claude 모델 사용
- Anthropic 계정 필요 (무료 한도 있음)

### Codex CLI

```bash
# 설치
brew install codex
# 또는
cargo install codex-cli

# freeapi 내부 호출
codex exec --full-auto <prompt>
```

- OpenAI 모델 사용
- OpenAI 계정 + 크레딧 필요

### Copilot CLI

```bash
# 설치
# 공식 설치 가이드 참고

# freeapi 내부 호출
copilot -p <prompt> --allow-all-tools
```

- GitHub Copilot 구독 필요

### OpenCode

```bash
# 설치
curl -fsSL https://opencode.ai/install | bash

# freeapi 내부 호출
opencode run <prompt>
```

---

## API Provider

HTTP API를 직접 호출합니다. API 키가 필요하지만 더 빠르고 안정적입니다.

### Google Gemini API

| 항목 | 값 |
|------|-----|
| 엔드포인트 | `generativelanguage.googleapis.com/v1beta` |
| 기본 모델 | `gemini-2.5-flash` |
| 무료 한도 | 10 RPM, 250 RPD, 250K TPM |
| 환경변수 | `GEMINI_API_KEY` 또는 `GOOGLE_API_KEY` |
| 키 발급 | https://ai.google.dev |
| 컨텍스트 | 최대 1M 토큰 |

사용 가능한 모델:
```
gemini-2.5-flash       (기본, 10 RPM, 250 RPD)
gemini-2.5-flash-lite  (15 RPM, 1000 RPD)
gemini-2.0-flash       (5 RPM, 100 RPD)
gemini-2.5-pro         (5 RPM, 100 RPD)
gemini-1.5-flash       (15 RPM, 1500 RPD)
```

### Groq

| 항목 | 값 |
|------|-----|
| 엔드포인트 | `api.groq.com/openai/v1` (OpenAI 호환) |
| 기본 모델 | `llama-3.3-70b-versatile` |
| 무료 한도 | 30 RPM, 1,000 RPD, 12K TPM |
| 환경변수 | `GROQ_API_KEY` |
| 키 발급 | https://console.groq.com/keys |
| 특징 | 300+ tok/s, 가장 빠른 무료 추론 |

사용 가능한 모델:
```
llama-3.3-70b-versatile (기본, 30 RPM)
llama-3.1-8b-instant    (30 RPM, 14400 RPD)
llama-4-scout-17b-16e-instruct (30 RPM)
qwen-qwq-32b           (30 RPM)
```

### Cerebras

| 항목 | 값 |
|------|-----|
| 엔드포인트 | `api.cerebras.ai/v1` (OpenAI 호환) |
| 기본 모델 | `llama-3.3-70b` |
| 무료 한도 | 30 RPM, 14,400 RPD, 60K TPM, 1M tok/day |
| 환경변수 | `CEREBRAS_API_KEY` |
| 키 발급 | https://cloud.cerebras.ai |
| 특징 | 가장 많은 일일 무료 토큰 |

### Mistral

| 항목 | 값 |
|------|-----|
| 엔드포인트 | `api.mistral.ai/v1` (OpenAI 호환) |
| 기본 모델 | `mistral-small-latest` |
| 무료 한도 | 1 RPS, 500K TPM, 10억 tok/월 |
| 환경변수 | `MISTRAL_API_KEY` |
| 키 발급 | https://console.mistral.ai/api-keys |
| 특징 | 월 토큰 한도가 가장 큼 |

### OpenRouter

| 항목 | 값 |
|------|-----|
| 엔드포인트 | `openrouter.ai/api/v1` (OpenAI 호환) |
| 기본 모델 | `deepseek/deepseek-r1:free` |
| 무료 한도 | 20 RPM, 50 RPD |
| 환경변수 | `OPENROUTER_API_KEY` |
| 키 발급 | https://openrouter.ai/keys |
| 특징 | 27+ 무료 모델, 모델 다양성 최고 |

무료 모델 예시:
```
deepseek/deepseek-r1:free
qwen/qwen3-coder-480b:free
meta-llama/llama-3.3-70b-instruct:free
google/gemma-3-27b-it:free
mistralai/mistral-small-3.1-24b-instruct:free
```

### Cohere

| 항목 | 값 |
|------|-----|
| 엔드포인트 | `api.cohere.ai/v2` |
| 기본 모델 | `command-r-plus` |
| 무료 한도 | 20 RPM, 1,000 요청/월 |
| 환경변수 | `COHERE_API_KEY` 또는 `CO_API_KEY` |
| 키 발급 | https://dashboard.cohere.com/api-keys |
| 스트리밍 | SSE 스트리밍 지원 |

### GitHub Models

| 항목 | 값 |
|------|-----|
| 엔드포인트 | `models.inference.ai.azure.com` (OpenAI 호환) |
| 기본 모델 | `gpt-4o` |
| 무료 한도 | 10 RPM, 50 RPD |
| 환경변수 | `GITHUB_TOKEN` |
| 키 발급 | `gh auth token` (GitHub CLI) 또는 https://github.com/settings/tokens |
| 특징 | GPT-4o, GPT-5 등 프리미엄 모델 무료 테스트 |

### Cloudflare Workers AI

| 항목 | 값 |
|------|-----|
| 엔드포인트 | `api.cloudflare.com/client/v4/accounts/{account_id}/ai/v1` (OpenAI 호환) |
| 기본 모델 | `@cf/meta/llama-3.3-70b-instruct-fp8-fast` |
| 환경변수 | `CLOUDFLARE_API_TOKEN` + `CLOUDFLARE_ACCOUNT_ID` |
| 키 발급 | https://dash.cloudflare.com/profile/api-tokens |
| 특징 | Cloudflare 계정의 Workers AI 모델을 직접 fallback 체인에 포함 가능 |

사용 가능한 모델 예시:
```
@cf/meta/llama-3.3-70b-instruct-fp8-fast
@cf/meta/llama-3.1-8b-instruct
@cf/openai/gpt-oss-120b
```

---

## Rate Limit 감지

### HTTP 429 감지
모든 API provider는 HTTP 429 응답 시 자동으로 rate limited로 표시됩니다.

파싱하는 헤더:
- `Retry-After` (초 단위)
- `X-Ratelimit-Remaining-Requests`
- `X-Ratelimit-Reset-Requests`

기본 대기 시간: 60초

### CLI Quota 감지
CLI provider의 stderr/stdout에서 키워드로 감지:
- `rate limit`, `quota exceeded`, `too many requests`
- `429`, `resource exhausted`, `throttl`

감지 시 해당 CLI를 60초간 비활성화하고 다음 provider로 전환합니다.

## 멀티턴 컨텍스트

API provider는 role 기반 메시지 배열을 그대로 전달합니다.

CLI provider는 대화 히스토리를 transcript prompt로 합성해서 전달합니다:
- `System instructions:`
- `Conversation:`
- `User:` / `Assistant:` 순서
