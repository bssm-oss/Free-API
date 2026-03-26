package cmd

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var spinnerOutput io.Writer = os.Stderr
var spinnerInterval = 120 * time.Millisecond
var spinnerFrames = []string{"|", "/", "-", "\\"}
var spinnerIsInteractive = func() bool {
	stat, err := os.Stderr.Stat()
	return err == nil && (stat.Mode()&os.ModeCharDevice) != 0
}

type terminalSpinner struct {
	enabled bool
	stopCh  chan struct{}
	doneCh  chan struct{}
	once    sync.Once
}

func startRequestSpinner(enabled bool, message string) *terminalSpinner {
	if !enabled {
		return &terminalSpinner{}
	}

	s := &terminalSpinner{
		enabled: true,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	go func() {
		defer close(s.doneCh)

		ticker := time.NewTicker(spinnerInterval)
		defer ticker.Stop()

		frame := 0
		for {
			fmt.Fprintf(spinnerOutput, "\r%s %s", spinnerFrames[frame%len(spinnerFrames)], message)
			frame++

			select {
			case <-ticker.C:
			case <-s.stopCh:
				fmt.Fprint(spinnerOutput, "\r\033[K")
				return
			}
		}
	}()

	return s
}

func (s *terminalSpinner) Stop() {
	if s == nil || !s.enabled {
		return
	}

	s.once.Do(func() {
		close(s.stopCh)
		<-s.doneCh
	})
}

func shouldShowSpinner(raw bool) bool {
	return !raw && spinnerIsInteractive()
}
