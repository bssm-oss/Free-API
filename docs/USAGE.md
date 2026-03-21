# 사용법 가이드

## 빠른 시작

```bash
# 설치
go install github.com/bssm-oss/Free-API@latest

# AI CLI가 하나라도 설치되어 있으면 바로 사용 가능
freeapi "안녕하세요"
```

## 채팅

### 기본 채팅

```bash
freeapi "Go의 장점을 설명해줘"
# 또는
freeapi chat "Go의 장점을 설명해줘"
```

### 대화 이어가기

```bash
freeapi "수도가 어디야?"
# → 서울입니다.

freeapi chat -c "인구는?"
# → (이전 컨텍스트를 기억하고) 서울의 인구는 약 950만명입니다.

freeapi chat -c "면적은?"
# → 서울의 면적은 약 605km²입니다.
```

### 특정 대화 이어가기

```bash
freeapi history list
# a1b2c3d4  2024-01-15 14:30  [4 msgs]  수도가 어디야?

freeapi chat --cid a1b2c3d4 "GDP는?"
```

### Provider 지정

```bash
# CLI provider
freeapi chat -p gemini-cli "hello"
freeapi chat -p claude-cli "hello"

# API provider (키 필요)
freeapi chat -p groq "hello"
freeapi chat -p gemini "hello"
```

### 모델 지정

```bash
freeapi chat -m gemini-2.5-pro "복잡한 질문"
freeapi chat -m llama-3.1-8b-instant "간단한 질문"
```

### 시스템 프롬프트

```bash
freeapi chat -s "한국어로만 답해줘" "What is Go?"
freeapi chat -s "코드만 답해줘, 설명 불필요" "Python quicksort"
```

### 파이프 입력

```bash
# 파일 분석
cat error.log | freeapi chat "이 에러 분석해줘"

# Git diff 리뷰
git diff | freeapi chat "이 변경사항 리뷰해줘"

# 코드 설명
cat main.go | freeapi chat "이 코드 설명해줘"

# 파이프 + 추가 지시
cat data.json | freeapi chat "이 데이터를 요약 테이블로 만들어줘"
```

### 출력 제어

```bash
# 메타데이터 없이 순수 응답만 (파이프/스크립트용)
freeapi chat --raw "답만 줘" > output.txt

# 스트리밍 없이 한번에 출력
freeapi chat --no-stream "질문"

# 타임아웃 설정 (초)
freeapi chat --timeout 30 "빠르게 답해줘"
```

## REPL (인터랙티브 모드)

```bash
freeapi    # 인자 없이 실행하면 REPL 진입
```

```
💬 freeapi REPL. Type /help for commands, Ctrl+C to exit.

you> 안녕하세요
AI> 안녕하세요! 무엇을 도와드릴까요?
[gemini-cli]

you> Go로 hello world 짜줘
AI> ...
[gemini-cli]

you> /new
📝 New conversation started.

you> 다른 주제로 대화
AI> ...

you> /history
→ a1b2c3d4  [2 msgs]  다른 주제로 대화
  e5f6g7h8  [4 msgs]  안녕하세요

you> /last
📎 Switched to conversation [e5f6g7h8]

you> /quit
Bye!
```

### REPL 명령어

| 명령어 | 설명 |
|--------|------|
| `/new` | 새 대화 시작 |
| `/last` | 이전 대화로 전환 |
| `/history` | 최근 대화 목록 (→ 현재 대화) |
| `/status` | provider 상태 확인 |
| `/id` | 현재 대화 ID |
| `/help` | 도움말 |
| `/quit` | 종료 |

## 대화 관리

```bash
# 최근 대화 목록
freeapi history list
freeapi history list -n 50    # 50개까지

# 대화 내용 보기
freeapi history show a1b2c3d4

# 대화 삭제
freeapi history delete a1b2c3d4

# 전체 삭제
freeapi history clear

# 마크다운으로 내보내기
freeapi export a1b2c3d4 > conversation.md
freeapi export a1b2c3d4 --format text > conversation.txt
```

## Provider 관리

```bash
# 전체 목록 (CLI + API)
freeapi providers list

# 상세 상태
freeapi providers status

# 연결 테스트
freeapi providers test

# 설치된 AI CLI 스캔
freeapi scan

# 사용 가능한 모델 목록
freeapi models
```

## 설정

### 설정 위자드

```bash
freeapi setup
```

CLI 자동감지 → API 키 설정까지 안내합니다.

### API 키 설정

```bash
# 환경변수 (추천)
export GEMINI_API_KEY="AIza..."
export GROQ_API_KEY="gsk_..."
export GITHUB_TOKEN="$(gh auth token)"
export CLOUDFLARE_API_TOKEN="..."
export CLOUDFLARE_ACCOUNT_ID="..."

# 또는 config 명령
freeapi config set gemini.api_key "AIza..."
freeapi config set groq.api_key "gsk_..."
freeapi config set cloudflare.account_id "..."
```

### 기타 설정

```bash
# 기본 시스템 프롬프트
freeapi config set default_system_prompt "한국어로 답해줘"

# 모델 변경
freeapi config set gemini.model "gemini-2.5-pro"
freeapi config set groq.model "llama-3.1-8b-instant"

# 컨텍스트 크기
freeapi config set max_context_messages 100

# CLI provider 비활성화
freeapi config set codex-cli.enabled false

# 현재 설정 확인
freeapi config list
```

### 설정 파일 위치

- 설정: `~/.config/freeapi/config.yaml`
- 대화 DB: `~/.local/share/freeapi/conversations.db`

## 스크립트 활용 예시

```bash
# 매일 뉴스 요약
curl -s https://news.api/today | freeapi chat --raw "3줄 요약"

# Git 커밋 메시지 생성
git diff --staged | freeapi chat --raw "커밋 메시지 작성해줘"

# 코드 리뷰 자동화
for f in $(git diff --name-only); do
  echo "=== $f ==="
  cat "$f" | freeapi chat --raw "버그가 있으면 알려줘"
done

# 번역
echo "Hello world" | freeapi chat --raw "한국어로 번역"
```
