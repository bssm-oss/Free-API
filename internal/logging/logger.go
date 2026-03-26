package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const maxLogSizeBytes int64 = 10 * 1024 * 1024

type level int

const (
	levelError level = iota
	levelInfo
	levelDebug
)

type logger struct {
	mu    sync.Mutex
	path  string
	level level
}

var global = &logger{}

func Configure(path, logLevel string) {
	global.mu.Lock()
	defer global.mu.Unlock()
	global.path = resolveLogPath(path)
	global.level = parseLevel(logLevel)
}

func Path() string {
	global.mu.Lock()
	defer global.mu.Unlock()
	if global.path == "" {
		global.path = resolveLogPath("")
	}
	return global.path
}

func Error(event string, fields map[string]any) {
	logEvent(levelError, "error", event, fields)
}

func Info(event string, fields map[string]any) {
	logEvent(levelInfo, "info", event, fields)
}

func Debug(event string, fields map[string]any) {
	logEvent(levelDebug, "debug", event, fields)
}

func logEvent(entryLevel level, levelName, event string, fields map[string]any) {
	global.mu.Lock()
	defer global.mu.Unlock()

	if global.path == "" {
		global.path = resolveLogPath("")
	}
	if entryLevel > global.level {
		return
	}

	entry := map[string]any{
		"ts":    time.Now().Format(time.RFC3339Nano),
		"level": levelName,
		"event": event,
		"pid":   os.Getpid(),
	}
	for k, v := range fields {
		entry[k] = v
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	data = append(data, '\n')

	if err := ensureLogFile(global.path); err != nil {
		return
	}
	if err := rotateIfNeeded(global.path); err != nil {
		return
	}

	f, err := os.OpenFile(global.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(data)
}

func ensureLogFile(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func rotateIfNeeded(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Size() < maxLogSizeBytes {
		return nil
	}

	rotated := path + "." + time.Now().Format("20060102-150405")
	return os.Rename(path, rotated)
}

func resolveLogPath(path string) string {
	if trimmed := strings.TrimSpace(path); trimmed != "" {
		return filepath.Clean(trimmed)
	}
	if envPath := strings.TrimSpace(os.Getenv("FREEAPI_LOG_PATH")); envPath != "" {
		return filepath.Clean(envPath)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "freeapi", "logs", "freeapi.log")
}

func parseLevel(value string) level {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(os.Getenv("FREEAPI_LOG_LEVEL")))
	}
	switch value {
	case "debug":
		return levelDebug
	case "error":
		return levelError
	default:
		return levelInfo
	}
}
