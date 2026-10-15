package cli

import (
	"fmt"
	"math"
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

// run animates a true-color gradient shimmer over "Thinking...", the same
// style as Anthropic/OpenCode (and vinhnx/tui-shimmer): a bright band sweeps
// left-to-right through the dim word every ~2s and loops.
func (s *spinner) run() {
	defer close(s.doneCh)
	ticker := time.NewTicker(33 * time.Millisecond)
	defer ticker.Stop()
	word := []rune(shimmerWord)
	defer fmt.Fprintf(os.Stderr, "\r\033[2K")
	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			elapsed := time.Since(s.start)
			phase := math.Mod(elapsed.Seconds()/2.0, 1.0)
			line := shimmerRender(word, phase, elapsed.Seconds())
			fmt.Fprintf(os.Stderr, "\r%s", line)
		}
	}
}

// shimmerRender builds an ANSI 24-bit-color shimmer frame.
func shimmerRender(word []rune, phase, elapsed float64) string {
	// base (dim) and peak (bright) RGB.
	bR, bG, bB := 60.0, 60.0, 70.0
	pR, pG, pB := 255.0, 255.0, 255.0

	n := len(word)
	// peak centre travels off-screen left -> off-screen right across the cycle
	peakPos := phase*float64(n+8) - 4.0

	var b strings.Builder
	for j := 0; j < n; j++ {
		d := math.Abs(float64(j) - peakPos)
		// bell-shaped intensity, radius ~2.2 chars
		var intensity float64
		if d < 2.2 {
			intensity = math.Exp(-(d * d) / (2 * 1.0 * 1.0))
		}
		if intensity < 0 {
			intensity = 0
		}
		if intensity > 1 {
			intensity = 1
		}
		r := bR + (pR-bR)*intensity
		g := bG + (pG-bG)*intensity
		bl := bB + (pB-bB)*intensity
		fmt.Fprintf(&b, "\033[38;2;%d;%d;%dm", int(r), int(g), int(bl))
		b.WriteRune(word[j])
	}
	// elapsed in dim gray after the word
	fmt.Fprintf(&b, "\033[38;2;80;80;90m  %.1fs\033[0m", elapsed)
	return b.String()
}

func startSpinner(message string) func() {
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
