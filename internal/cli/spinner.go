package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

type spinner struct {
	message string
	start   time.Time
	stopCh  chan struct{}
	doneCh  chan struct{}
}

func newSpinner(message string) *spinner {
	return &spinner{
		message: message,
		start:   time.Now(),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

func (s *spinner) run() {
	defer close(s.doneCh)
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()
	i := 0
	lastWidth := 0
	defer fmt.Fprintf(os.Stderr, "\r\033[2K")
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			elapsed := time.Since(s.start).Seconds()
			frame := spinnerFrames[i%len(spinnerFrames)]
			line := fmt.Sprintf("%s  %s  %.1fs", frame, s.message, elapsed)
			width := len([]rune(line))
			pad := 0
			if width < lastWidth {
				pad = lastWidth - width
			}
			fmt.Fprintf(os.Stderr, "\r%s%s", line, strings.Repeat(" ", pad))
			lastWidth = width
			i++
		}
	}
}

func startSpinner(message string) func() {
	if os.Getenv("NO_COLOR") != "" || !term.IsTerminal(int(os.Stderr.Fd())) {
		return func() {}
	}
	sp := newSpinner(message)
	go sp.run()
	var once sync.Once
	return func() {
		once.Do(func() { close(sp.stopCh) })
		<-sp.doneCh
	}
}
