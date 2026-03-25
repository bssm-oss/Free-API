package cmd

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var helpLang string

type commandCopy struct {
	Use     string
	Short   string
	Long    string
	Example string
}

type localizedCommand struct {
	Use     string
	Short   string
	Long    string
	Example string
}

type helpLabels struct {
	Usage             string
	AvailableCommands string
	Flags             string
	GlobalFlags       string
	Examples          string
	MoreInfo          string
}

var (
	helpI18nOnce      sync.Once
	originalCommands  = map[*cobra.Command]commandCopy{}
	originalFlagUsage = map[*cobra.Command]map[string]string{}
)

var helpLabelCatalog = map[string]helpLabels{
	"en": {
		Usage:             "Usage",
		AvailableCommands: "Available Commands",
		Flags:             "Flags",
		GlobalFlags:       "Global Flags",
		Examples:          "Examples",
		MoreInfo:          "Use %q for more information about a command.",
	},
	"ko": {
		Usage:             "사용법",
		AvailableCommands: "사용 가능한 명령어",
		Flags:             "플래그",
		GlobalFlags:       "전역 플래그",
		Examples:          "예시",
		MoreInfo:          "특정 명령어의 자세한 도움말은 %q 를 실행하세요.",
	},
}

var localizedCommands = map[string]localizedCommand{
	"freeapi": {
		Short: "무료 LLM 라우팅 CLI",
		Long: `freeapi - 여러 무료 LLM과 설치된 AI CLI를 하나로 묶는 도구입니다.

한 provider가 실패하거나 rate limit에 걸리면 다음 provider로 자동 전환합니다.
Gemini, Groq, Cerebras, Mistral, OpenRouter, Cohere, GitHub Models, Cloudflare와 설치된 AI CLI를 지원합니다.

대화 컨텍스트는 SQLite에 저장되어 멀티턴 대화를 자연스럽게 이어갈 수 있습니다.

빠른 시작:
  freeapi "Go interface를 설명해줘"   # 바로 질문
  freeapi chat -c "조금 더 자세히"     # 이전 대화 이어가기
  freeapi                              # 인터랙티브 REPL`,
	},
	"freeapi help": {
		Short: "명령어 도움말 보기",
	},
	"freeapi completion": {
		Short: "지정한 셸의 자동완성 스크립트 생성",
	},
	"freeapi completion bash": {
		Short: "bash 자동완성 스크립트 생성",
	},
	"freeapi completion fish": {
		Short: "fish 자동완성 스크립트 생성",
	},
	"freeapi completion powershell": {
		Short: "PowerShell 자동완성 스크립트 생성",
	},
	"freeapi completion zsh": {
		Short: "zsh 자동완성 스크립트 생성",
	},
	"freeapi chat": {
		Short: "LLM에게 메시지 보내기",
		Long: `메시지를 보내고 응답을 받습니다. 사용 가능한 무료 provider 중 가장 적절한 것을 자동으로 선택합니다.

예시:
  freeapi chat "Go interface를 설명해줘"
  freeapi chat -c "조금 더 자세히"
  freeapi chat --provider gemini "hello"
  freeapi chat --raw "응답만 줘" > output.txt
  echo "analyze this" | freeapi chat`,
	},
	"freeapi config": {
		Short: "freeapi 설정 관리",
	},
	"freeapi config list": {
		Short: "현재 설정 보기",
	},
	"freeapi config set": {
		Short: "설정값 변경",
		Long: `설정값을 변경합니다. 중첩 키는 점 표기법으로 지정합니다.

예시:
  freeapi config set gemini.api_key "AIza..."
  freeapi config set groq.api_key "gsk_..."
  freeapi config set gemini.enabled false
  freeapi config set codex-cli.enabled false
  freeapi config set max_context_messages 100
  freeapi config set default_system_prompt "You are a coding assistant."`,
	},
	"freeapi config init": {
		Short: "기본 설정 파일 생성",
	},
	"freeapi export": {
		Short: "대화를 markdown 또는 text로 내보내기",
		Long: `대화를 stdout으로 내보냅니다. --format 으로 출력 형식을 선택합니다.

예시:
  freeapi export abc123                    # markdown으로 내보내기
  freeapi export abc123 --format text      # plain text로 내보내기
  freeapi export abc123 > conversation.md  # 파일로 저장`,
	},
	"freeapi history": {
		Short: "대화 기록 관리",
	},
	"freeapi history list": {
		Short: "최근 대화 목록 보기",
	},
	"freeapi history show": {
		Short: "대화 내용 보기",
	},
	"freeapi history delete": {
		Short: "대화 하나 삭제",
	},
	"freeapi history clear": {
		Short: "모든 대화 삭제",
	},
	"freeapi models": {
		Short: "provider별 사용 가능한 모델 보기",
	},
	"freeapi providers": {
		Short: "LLM provider 관리",
	},
	"freeapi providers list": {
		Short: "설정된 provider 목록 보기",
	},
	"freeapi providers status": {
		Short: "provider 상태 자세히 보기",
	},
	"freeapi providers test": {
		Short: "모든 provider 연결 테스트",
	},
	"freeapi scan": {
		Short: "설치된 AI CLI 스캔",
	},
	"freeapi server": {
		Short: "로컬 HTTP 서버 시작",
		Long: `freeapi용 로컬 HTTP 서버를 시작합니다.

엔드포인트:
  POST /freeapi/chat   JSON body: {"message":"hello"}
  GET  /healthz        상태 확인`,
	},
	"freeapi setup": {
		Short: "CLI 자동 감지와 API 키 설정",
	},
	"freeapi version": {
		Short: "버전 정보 출력",
	},
}

var localizedFlagUsage = map[string]map[string]string{
	"freeapi": {
		"lang": "도움말 언어 (en, ko)",
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi chat": {
		"continue":  "마지막 대화를 이어서 사용",
		"cid":       "특정 대화 ID로 이어서 사용",
		"provider":  "특정 provider 사용",
		"model":     "특정 모델 사용",
		"system":    "system message 지정",
		"no-stream": "스트리밍 출력 비활성화",
		"raw":       "메타데이터 없이 응답만 출력",
		"timeout":   "요청 타임아웃(초)",
		"help":      "이 명령어의 도움말 보기",
	},
	"freeapi config": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi config list": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi config set": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi config init": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi export": {
		"format": "출력 형식: markdown, text",
		"help":   "이 명령어의 도움말 보기",
	},
	"freeapi history": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi history list": {
		"limit": "표시할 대화 개수",
		"help":  "이 명령어의 도움말 보기",
	},
	"freeapi history show": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi history delete": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi history clear": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi models": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi providers": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi providers list": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi providers status": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi providers test": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi scan": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi server": {
		"addr": "HTTP 리슨 주소",
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi setup": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi version": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi help": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi completion": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi completion bash": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi completion fish": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi completion powershell": {
		"help": "이 명령어의 도움말 보기",
	},
	"freeapi completion zsh": {
		"help": "이 명령어의 도움말 보기",
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&helpLang, "lang", "", "Help language (en, ko)")
}

func configureLocalizedHelp(args []string) {
	initHelpI18n()

	lang := resolveHelpLanguage(args)
	restoreOriginalHelpText(rootCmd)
	applyLocalizedCommandText(rootCmd, lang)
	applyLocalizedFlagText(rootCmd, lang)
	applyLocalizedTemplates(rootCmd, lang)
}

func initHelpI18n() {
	helpI18nOnce.Do(func() {
		rootCmd.InitDefaultHelpCmd()
		rootCmd.InitDefaultCompletionCmd()
		walkCommands(rootCmd, func(cmd *cobra.Command) {
			cmd.InitDefaultHelpFlag()
			originalCommands[cmd] = commandCopy{
				Use:     cmd.Use,
				Short:   cmd.Short,
				Long:    cmd.Long,
				Example: cmd.Example,
			}
			originalFlagUsage[cmd] = snapshotFlagUsage(cmd)
		})
	})
}

func snapshotFlagUsage(cmd *cobra.Command) map[string]string {
	usages := map[string]string{}
	collectFlagUsage := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(flag *pflag.Flag) {
			usages[flag.Name] = flag.Usage
		})
	}

	collectFlagUsage(cmd.LocalFlags())
	collectFlagUsage(cmd.InheritedFlags())
	collectFlagUsage(cmd.PersistentFlags())
	return usages
}

func resolveHelpLanguage(args []string) string {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--lang" && i+1 < len(args) {
			return normalizeHelpLanguage(args[i+1])
		}
		if strings.HasPrefix(arg, "--lang=") {
			return normalizeHelpLanguage(strings.TrimPrefix(arg, "--lang="))
		}
	}

	for _, env := range []string{"FREEAPI_LANG", "LC_ALL", "LANG"} {
		if value := strings.TrimSpace(os.Getenv(env)); value != "" {
			return normalizeHelpLanguage(value)
		}
	}

	return "en"
}

func normalizeHelpLanguage(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch {
	case value == "":
		return "en"
	case strings.HasPrefix(value, "ko"), strings.Contains(value, "korean"), strings.Contains(value, "한국어"):
		return "ko"
	case strings.HasPrefix(value, "en"), strings.Contains(value, "english"):
		return "en"
	default:
		return "en"
	}
}

func restoreOriginalHelpText(cmd *cobra.Command) {
	walkCommands(cmd, func(current *cobra.Command) {
		if original, ok := originalCommands[current]; ok {
			current.Use = original.Use
			current.Short = original.Short
			current.Long = original.Long
			current.Example = original.Example
		}

		if usages, ok := originalFlagUsage[current]; ok {
			restoreFlagUsage(current.LocalFlags(), usages)
			restoreFlagUsage(current.InheritedFlags(), usages)
			restoreFlagUsage(current.PersistentFlags(), usages)
		}
	})
}

func restoreFlagUsage(fs *pflag.FlagSet, usages map[string]string) {
	fs.VisitAll(func(flag *pflag.Flag) {
		if usage, ok := usages[flag.Name]; ok {
			flag.Usage = usage
		}
	})
}

func applyLocalizedCommandText(cmd *cobra.Command, lang string) {
	if lang != "ko" {
		return
	}

	walkCommands(cmd, func(current *cobra.Command) {
		override, ok := localizedCommands[current.CommandPath()]
		if !ok {
			return
		}
		if override.Use != "" {
			current.Use = override.Use
		}
		if override.Short != "" {
			current.Short = override.Short
		}
		if override.Long != "" {
			current.Long = override.Long
		}
		if override.Example != "" {
			current.Example = override.Example
		}
	})
}

func applyLocalizedFlagText(cmd *cobra.Command, lang string) {
	if lang != "ko" {
		return
	}

	walkCommands(cmd, func(current *cobra.Command) {
		usages, ok := localizedFlagUsage[current.CommandPath()]
		if !ok {
			return
		}
		applyFlagUsage(current.LocalFlags(), usages)
		applyFlagUsage(current.InheritedFlags(), usages)
		applyFlagUsage(current.PersistentFlags(), usages)
	})
}

func applyFlagUsage(fs *pflag.FlagSet, usages map[string]string) {
	fs.VisitAll(func(flag *pflag.Flag) {
		if usage, ok := usages[flag.Name]; ok {
			flag.Usage = usage
		}
	})
}

func applyLocalizedTemplates(cmd *cobra.Command, lang string) {
	labels, ok := helpLabelCatalog[lang]
	if !ok {
		labels = helpLabelCatalog["en"]
	}

	usageTemplate := buildUsageTemplate(labels)
	helpTemplate := "{{with (or .Long .Short)}}{{. | trimTrailingWhitespaces}}\n\n{{end}}{{if or .Runnable .HasAvailableSubCommands}}{{.UsageString}}{{end}}"

	walkCommands(cmd, func(current *cobra.Command) {
		current.SetUsageTemplate(usageTemplate)
		current.SetHelpTemplate(helpTemplate)
	})
}

func buildUsageTemplate(labels helpLabels) string {
	moreInfo := fmt.Sprintf(labels.MoreInfo, "{{.CommandPath}} [command] --help")
	return fmt.Sprintf(`%s:
{{if .Runnable}}  {{.UseLine}}
{{end}}{{if .HasAvailableSubCommands}}  {{.CommandPath}} [command]
{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

%s:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

%s:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

%s:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

%s:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

%s{{end}}
`, labels.Usage, labels.Examples, labels.AvailableCommands, labels.Flags, labels.GlobalFlags, moreInfo)
}

func walkCommands(cmd *cobra.Command, fn func(*cobra.Command)) {
	fn(cmd)
	for _, child := range cmd.Commands() {
		walkCommands(child, fn)
	}
}
