package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

func TestCLIConversationFlowEndToEnd(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("continue includes prior transcript for CLI provider", func(t *testing.T) {
		env, stateDir := newCLIEnv(t, map[string]string{
			"gemini": fakeCLIRecorderScript("gemini", "gemini"),
		})

		runCmd(t, env, bin, "chat", "--no-stream", "--raw", "first question")
		runCmd(t, env, bin, "chat", "--no-stream", "--raw", "-c", "follow up question")

		prompt := readFile(t, filepath.Join(stateDir, "gemini_prompt_2.txt"))
		for _, want := range []string{
			"User:\nfirst question",
			"Assistant:\ngemini-response-1",
			"User:\nfollow up question",
		} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("second prompt missing %q:\n%s", want, prompt)
			}
		}
	})

	t.Run("cid accepts short ID from history output", func(t *testing.T) {
		env, stateDir := newCLIEnv(t, map[string]string{
			"gemini": fakeCLIRecorderScript("gemini", "gemini"),
		})

		runCmd(t, env, bin, "chat", "--no-stream", "--raw", "prefix base question")
		history := runCmd(t, env, bin, "history", "list")

		shortID := regexp.MustCompile(`[0-9a-f]{8}`).FindString(history)
		if shortID == "" {
			t.Fatalf("failed to parse short conversation id from history output:\n%s", history)
		}

		runCmd(t, env, bin, "chat", "--no-stream", "--raw", "--cid", shortID, "prefix follow up")

		prompt := readFile(t, filepath.Join(stateDir, "gemini_prompt_2.txt"))
		for _, want := range []string{
			"User:\nprefix base question",
			"Assistant:\ngemini-response-1",
			"User:\nprefix follow up",
		} {
			if !strings.Contains(prompt, want) {
				t.Fatalf("prompt missing %q:\n%s", want, prompt)
			}
		}
	})

	t.Run("cli disable and priority affect runtime selection", func(t *testing.T) {
		env, stateDir := newCLIEnv(t, map[string]string{
			"gemini": fakeCLIRecorderScript("gemini", "gemini"),
			"claude": fakeCLIRecorderScript("claude", "claude"),
		})

		runCmd(t, env, bin, "config", "set", "gemini-cli.enabled", "false")
		runCmd(t, env, bin, "chat", "--no-stream", "--raw", "routing check")

		if _, err := os.Stat(filepath.Join(stateDir, "gemini_prompt_1.txt")); !os.IsNotExist(err) {
			t.Fatalf("gemini should be disabled, but it was invoked")
		}
		if _, err := os.Stat(filepath.Join(stateDir, "claude_prompt_1.txt")); err != nil {
			t.Fatalf("expected claude to handle the request: %v", err)
		}

		runCmd(t, env, bin, "config", "set", "gemini-cli.enabled", "true")
		runCmd(t, env, bin, "config", "set", "claude-cli.priority", "5")
		runCmd(t, env, bin, "chat", "--no-stream", "--raw", "priority check")

		if _, err := os.Stat(filepath.Join(stateDir, "claude_prompt_2.txt")); err != nil {
			t.Fatalf("expected claude to be invoked again after priority override: %v", err)
		}
	})

	t.Run("default cli priority prefers codex before gemini and claude", func(t *testing.T) {
		env, stateDir := newCLIEnv(t, map[string]string{
			"codex":  fakeCLIRecorderScript("codex", "codex"),
			"gemini": fakeCLIRecorderScript("gemini", "gemini"),
			"claude": fakeCLIRecorderScript("claude", "claude"),
		})

		runCmd(t, env, bin, "chat", "--no-stream", "--raw", "priority default check")

		if _, err := os.Stat(filepath.Join(stateDir, "codex_prompt_1.txt")); err != nil {
			t.Fatalf("expected codex to handle the request first: %v", err)
		}
		if _, err := os.Stat(filepath.Join(stateDir, "gemini_prompt_1.txt")); !os.IsNotExist(err) {
			t.Fatalf("gemini should not run before codex by default")
		}
		if _, err := os.Stat(filepath.Join(stateDir, "claude_prompt_1.txt")); !os.IsNotExist(err) {
			t.Fatalf("claude should not run before codex by default")
		}
	})
}

func TestInteractiveAndDirectShareCLIPromptShape(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("first turn prompt matches", func(t *testing.T) {
		directEnv, directStateDir := newCLIEnv(t, map[string]string{
			"gemini": fakeCLIRecorderScript("gemini", "gemini"),
		})
		runCmd(t, directEnv, bin, "hello")
		directPrompt := readFile(t, filepath.Join(directStateDir, "gemini_prompt_1.txt"))

		replEnv, replStateDir := newCLIEnv(t, map[string]string{
			"gemini": fakeCLIRecorderScript("gemini", "gemini"),
		})
		runCmdWithInput(t, replEnv, bin, "hello\n/quit\n")
		replPrompt := readFile(t, filepath.Join(replStateDir, "gemini_prompt_1.txt"))

		if directPrompt != replPrompt {
			t.Fatalf("expected direct and repl prompts to match\n--- direct ---\n%s\n--- repl ---\n%s", directPrompt, replPrompt)
		}
	})

	t.Run("follow up prompt matches", func(t *testing.T) {
		directEnv, directStateDir := newCLIEnv(t, map[string]string{
			"gemini": fakeCLIRecorderScript("gemini", "gemini"),
		})
		runCmd(t, directEnv, bin, "hello")
		runCmd(t, directEnv, bin, "chat", "-c", "follow up")
		directPrompt := readFile(t, filepath.Join(directStateDir, "gemini_prompt_2.txt"))

		replEnv, replStateDir := newCLIEnv(t, map[string]string{
			"gemini": fakeCLIRecorderScript("gemini", "gemini"),
		})
		runCmdWithInput(t, replEnv, bin, "hello\nfollow up\n/quit\n")
		replPrompt := readFile(t, filepath.Join(replStateDir, "gemini_prompt_2.txt"))

		if directPrompt != replPrompt {
			t.Fatalf("expected direct and repl follow-up prompts to match\n--- direct ---\n%s\n--- repl ---\n%s", directPrompt, replPrompt)
		}
	})
}

func TestHelpLocalizationEndToEnd(t *testing.T) {
	bin := buildTestBinary(t)

	t.Run("root help supports Korean", func(t *testing.T) {
		env, _ := newCLIEnv(t, nil)
		out := runCmd(t, env, bin, "help", "--lang", "ko")

		for _, want := range []string{
			"freeapi - 여러 무료 LLM과 설치된 AI CLI를 하나로 묶는 도구입니다.",
			"사용법:",
			"사용 가능한 명령어:",
			"도움말 언어 (en, ko)",
			"특정 명령어의 자세한 도움말은",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("help output missing %q:\n%s", want, out)
			}
		}
	})

	t.Run("FREEAPI_LANG localizes subcommand help", func(t *testing.T) {
		env, _ := newCLIEnv(t, nil)
		env = append(env, "FREEAPI_LANG=ko")
		out := runCmd(t, env, bin, "chat", "--help")

		for _, want := range []string{
			"메시지를 보내고 응답을 받습니다.",
			"마지막 대화를 이어서 사용",
			"메타데이터 없이 응답만 출력",
			"요청 타임아웃(초)",
		} {
			if !strings.Contains(out, want) {
				t.Fatalf("localized chat help missing %q:\n%s", want, out)
			}
		}
	})
}

func TestUnknownCommandTypoShowsSuggestion(t *testing.T) {
	bin := buildTestBinary(t)
	env, _ := newCLIEnv(t, nil)

	out, err := runCmdExpectError(env, bin, "model")
	if err == nil {
		t.Fatal("expected typo command to fail")
	}

	for _, want := range []string{
		`unknown command "model"`,
		`Did you mean "models"?`,
		`freeapi chat "model"`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("error output missing %q:\n%s", want, out)
		}
	}
}

func buildTestBinary(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to locate test file")
	}
	rootDir := filepath.Dir(filepath.Dir(filename))
	bin := filepath.Join(t.TempDir(), "freeapi-test")

	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build test binary: %v\n%s", err, out)
	}
	return bin
}

func newCLIEnv(t *testing.T, scripts map[string]string) ([]string, string) {
	t.Helper()

	homeDir := t.TempDir()
	binDir := t.TempDir()
	stateDir := t.TempDir()

	for name, script := range scripts {
		path := filepath.Join(binDir, name)
		content := strings.ReplaceAll(script, "__STATE_DIR__", stateDir)
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			t.Fatalf("write fake cli %s: %v", name, err)
		}
	}

	pathEnv := strings.Join([]string{binDir, "/usr/bin", "/bin"}, string(os.PathListSeparator))

	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "HOME=") || strings.HasPrefix(kv, "PATH=") {
			continue
		}
		env = append(env, kv)
	}
	env = append(env, "HOME="+homeDir, "PATH="+pathEnv)

	return env, stateDir
}

func runCmd(t *testing.T, env []string, bin string, args ...string) string {
	t.Helper()

	out, err := runCmdExpectError(env, bin, args...)
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", bin, strings.Join(args, " "), err, out)
	}
	return out
}

func runCmdWithInput(t *testing.T, env []string, bin, input string, args ...string) string {
	t.Helper()

	out, err := runCmdWithInputExpectError(env, bin, input, args...)
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", bin, strings.Join(args, " "), err, out)
	}
	return out
}

func runCmdExpectError(env []string, bin string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func runCmdWithInputExpectError(env []string, bin, input string, args ...string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	cmd.Stdin = strings.NewReader(input)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func fakeCLIRecorderScript(name, binary string) string {
	return "#!/bin/sh\n" +
		"STATE_DIR=\"__STATE_DIR__\"\n" +
		"COUNT_FILE=\"$STATE_DIR/" + name + "_count\"\n" +
		"COUNT=0\n" +
		"if [ -f \"$COUNT_FILE\" ]; then COUNT=$(/bin/cat \"$COUNT_FILE\"); fi\n" +
		"COUNT=$((COUNT + 1))\n" +
		"printf '%s' \"$COUNT\" > \"$COUNT_FILE\"\n" +
		"PROMPT=\"\"\n" +
		"EXPECT_PROMPT=0\n" +
		"HAVE_PROMPT=0\n" +
		"for ARG in \"$@\"; do\n" +
		"  if [ \"$EXPECT_PROMPT\" = \"1\" ]; then PROMPT=\"$ARG\"; EXPECT_PROMPT=0; HAVE_PROMPT=1; continue; fi\n" +
		"  case \"$ARG\" in\n" +
		"    --prompt|-p)\n" +
		"      EXPECT_PROMPT=1\n" +
		"      ;;\n" +
		"    *)\n" +
		"      if [ \"$HAVE_PROMPT\" = \"0\" ]; then PROMPT=\"$ARG\"; fi\n" +
		"      ;;\n" +
		"  esac\n" +
		"done\n" +
		"PROMPT_FILE=\"$STATE_DIR/" + name + "_prompt_${COUNT}.txt\"\n" +
		"printf '%s' \"$PROMPT\" > \"$PROMPT_FILE\"\n" +
		"printf '" + binary + "-response-%s\\n' \"$COUNT\"\n"
}
