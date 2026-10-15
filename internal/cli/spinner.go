package cli

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

const shimmerWord = "Thinking..."

type spinner struct {
	start  time.Time
	stopCh chan struct{}
	doneCh chan struct{}
}

func newSpinner() *spinner {
	return &spinner{
		start:  time.Now(),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

// run animates a shimmer over "Thinking..." on stderr while waiting.
// Style mirrors the animated thinking indicator in Claude Code / opencode:
// a bright band sweeps left-to-right through the dim word, with elapsed time.
func (s *spinner) run() {
	defer close(s.doneCh)
	ticker := time.NewTicker(60 * time.Millisecond)
	defer ticker.Stop()
	word := []rune(shimmerWord)
	cycle := len(word) + 6 // +margin so the band fully exits before restarting
	i := 0
	lastWidth := 0
	defer fmt.Fprintf(os.Stderr, "\r\033[2K")
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			pos := i % cycle
			line := shimmerRender(word, pos, time.Since(s.start).Seconds())
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

// shimmerRender builds an ANSI-rendered shimmer frame for the given position.
func shimmerRender(word []rune, pos int, elapsed float64) string {
	const (
		dim    = "\033[90m"   // dim gray
		mid    = "\033[37m"   // light gray
		bright = "\033[1;97m" // bold white
		reset  = "\033[0m"
	)
	var b strings.Builder
	b.WriteString("\033[90m") // dim "⠋"? skip glyph
	for j := range word {
		d := j - pos + 2
		switch {
		case d == 2:
			b.WriteString(bright)
		case d == 1 || d == 3:
			b.WriteString(mid)
		default:
			b.WriteString(dim)
		}
		b.WriteRune(word[j])
	}
	b.WriteString(dim)
	fmt.Fprintf(&b, "  %.1fs", elapsed)
	b.WriteString(reset)
	return b.String()
}

func startSpinner(message string) func() {
	// message retained for backward compatibility; shimmer uses "Thinking...".
	_ = message
	if os.Getenv("NO_COLOR") != "" || !term.IsTerminal(int(os.Stderr.Fd())) {
		return func() {}
	}
	sp := newSpinner()
	go sp.run()
	var once sync.Once
	return func() {
		once.Do(func() { close(sp.stopCh) })
		<-sp.doneCh
	}
}
