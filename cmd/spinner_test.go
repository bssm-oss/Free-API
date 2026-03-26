package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestStartRequestSpinnerWritesFramesAndClears(t *testing.T) {
	var buf bytes.Buffer

	oldOutput := spinnerOutput
	oldInterval := spinnerInterval
	oldInteractive := spinnerIsInteractive
	defer func() {
		spinnerOutput = oldOutput
		spinnerInterval = oldInterval
		spinnerIsInteractive = oldInteractive
	}()

	spinnerOutput = &buf
	spinnerInterval = 5 * time.Millisecond
	spinnerIsInteractive = func() bool { return true }

	spinner := startRequestSpinner(shouldShowSpinner(false), "Waiting for response...")
	time.Sleep(20 * time.Millisecond)
	spinner.Stop()

	got := buf.String()
	if !strings.Contains(got, "Waiting for response...") {
		t.Fatalf("spinner output missing label: %q", got)
	}
	if !strings.Contains(got, "\r\033[K") {
		t.Fatalf("spinner output missing clear sequence: %q", got)
	}
}

func TestStartRequestSpinnerDisabledIsNoOp(t *testing.T) {
	var buf bytes.Buffer

	oldOutput := spinnerOutput
	defer func() {
		spinnerOutput = oldOutput
	}()

	spinnerOutput = &buf

	spinner := startRequestSpinner(false, "Waiting for response...")
	spinner.Stop()

	if got := buf.String(); got != "" {
		t.Fatalf("expected no spinner output, got %q", got)
	}
}
