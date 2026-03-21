# 개발 가이드

## 환경 설정

```bash
# Go 1.26.1+ 필요
go version

# 소스 클론
git clone https://github.com/bssm-oss/Free-API.git
cd Free-API
```

## 빌드

```bash
make build          # ./freeapi 바이너리 생성
make install        # ~/.local/bin/freeapi에 설치
make clean          # 바이너리 삭제
```

## 테스트

```bash
make test           # 전체 테스트
make vet            # 정적 분석

# 특정 패키지 테스트
go test ./internal/provider/ -v
go test ./internal/context/ -v
go test ./internal/config/ -v
```

현재 테스트:

| 패키지 | 테스트 수 | 내용 |
|--------|----------|------|
| `cmd` | 4 | config parsing, validation |
| `internal/provider` | 11 | rotator, CLI prompt/quota logic, Cloudflare |
| `internal/context` | 16 | store CRUD, prefix lookup, ordering, manager lifecycle |
| `internal/config` | 7 | defaults, env override, raw load |

## 크로스 컴파일

```bash
make cross          # 전체 플랫폼

# 개별
make cross-darwin   # macOS arm64 + amd64
make cross-linux    # Linux amd64 + arm64
make cross-windows  # Windows amd64
```

CGO 없이 pure Go이므로 크로스 컴파일이 바로 됩니다.

## 새 Provider 추가하기

### API Provider

1. `internal/provider/` 에 파일 생성 (예: `newprovider.go`)
2. `Provider` 인터페이스 구현:

```go
type NewProvider struct {
    apiKey       string
    baseURL      string
    defaultModel string
    mu           sync.Mutex
    rateLimit    models.RateLimitInfo
}

func (p *NewProvider) Name() string { return "newprovider" }
func (p *NewProvider) DefaultModel() string { return p.defaultModel }
func (p *NewProvider) IsAvailable() bool { ... }
func (p *NewProvider) Chat(ctx, messages, opts) (*Response, error) { ... }
func (p *NewProvider) ChatStream(ctx, messages, opts) (<-chan StreamChunk, error) { ... }
func (p *NewProvider) RateLimitStatus() RateLimitInfo { ... }
func (p *NewProvider) MarkRateLimited(info RateLimitInfo) { ... }
```

OpenAI 호환 API라면 `openai_compat.go`의 `NewOpenAICompat()`를 재사용:

```go
// registry.go의 createProvider에 추가
case "newprovider":
    return NewOpenAICompat("newprovider", cfg.APIKey, cfg.BaseURL, cfg.Model, nil)
```

3. `internal/config/config.go`의 `DefaultConfig()`에 provider 추가
4. `EnvVarMap`에 환경변수 매핑 추가

### CLI Provider

1. `internal/provider/cli_provider.go`의 `KnownCLIs()`에 추가:

```go
{
    Name: "newtool-cli",
    Args: func(prompt string) []string {
        return []string{"--flag", prompt}
    },
},
```

2. `BinNames`에 바이너리 이름 매핑:

```go
"newtool-cli": "newtool",
```

3. `cmd/scan.go`의 목록에 추가

## 코드 구조 규칙

- **`internal/`**: 외부 패키지에서 import 불가 (Go 규칙)
- **`cmd/`**: CLI 명령만, 비즈니스 로직 없음
- **Provider 인터페이스**: 모든 provider는 동일한 인터페이스
- **에러 처리**: `RateLimitError`는 로테이션 트리거, 기타 에러도 다음 provider로
- **트랜잭션**: 다수 DB 작업은 반드시 트랜잭션 사용
- **HTTP**: Chat은 `SharedClient`, Stream은 `StreamClient` 사용
- **Race condition**: `rateLimit` 필드는 mutex로 보호, 외부에서 직접 읽지 않음

## 의존성

| 패키지 | 용도 |
|--------|------|
| `github.com/spf13/cobra` | CLI 프레임워크 |
| `gopkg.in/yaml.v3` | YAML 설정 파일 |
| `modernc.org/sqlite` | Pure Go SQLite (CGO 불필요) |

직접 의존성 3개, 간접 의존성은 Go 표준 수준입니다.
