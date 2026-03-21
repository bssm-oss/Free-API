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

	pathEnv := binDir
	if existing := os.Getenv("PATH"); existing != "" {
		pathEnv += string(os.PathListSeparator) + existing
	}

	env := os.Environ()
	env = append(env, "HOME="+homeDir, "PATH="+pathEnv)

	return env, stateDir
}

func runCmd(t *testing.T, env []string, bin string, args ...string) string {
	t.Helper()

	cmd := exec.Command(bin, args...)
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", bin, strings.Join(args, " "), err, out)
	}
	return string(out)
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
		"if [ -f \"$COUNT_FILE\" ]; then COUNT=$(cat \"$COUNT_FILE\"); fi\n" +
		"COUNT=$((COUNT + 1))\n" +
		"printf '%s' \"$COUNT\" > \"$COUNT_FILE\"\n" +
		"PROMPT_FILE=\"$STATE_DIR/" + name + "_prompt_${COUNT}.txt\"\n" +
		"printf '%s' \"$2\" > \"$PROMPT_FILE\"\n" +
		"printf '" + binary + "-response-%s\\n' \"$COUNT\"\n"
}
