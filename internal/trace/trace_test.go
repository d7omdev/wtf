package trace

import (
	"strings"
	"testing"
)

func TestCleanStripsANSI(t *testing.T) {
	in := "\x1b[31merror\x1b[0m: file \x1b[1mmissing\x1b[0m"
	if got := strings.TrimSpace(Clean(in)); got != "error: file missing" {
		t.Errorf("Clean = %q", got)
	}
}

func TestCleanCarriageReturnOverwrite(t *testing.T) {
	in := "10%\r50%\r100%\nnext\n"
	got := Clean(in)
	if !strings.Contains(got, "100%") || strings.Contains(got, "10%") {
		t.Errorf("CR overwrite not resolved: %q", got)
	}
	if !strings.Contains(got, "next") {
		t.Errorf("lost following line: %q", got)
	}
}

func TestCleanCRLFAndControls(t *testing.T) {
	got := Clean("a\r\nb\x07\x00c\ttab\n")
	want := "a\nbc\ttab\n"
	if got != want {
		t.Errorf("Clean = %q, want %q", got, want)
	}
}

func TestCutAboveCommand(t *testing.T) {
	raw := "user@host:~$ npm test\nnpm warn ...\n> fail\nerror TS2304\nuser@host:~$ "
	got := CutAboveCommand(raw, "npm test")
	if strings.Contains(got, "user@host") {
		t.Errorf("prompt line not cut: %q", got)
	}
	if !strings.Contains(got, "error TS2304") {
		t.Errorf("output lost: %q", got)
	}
}

func TestCutAboveCommandNoMatchKeepsAll(t *testing.T) {
	raw := "no command echo here\nsome output"
	if got := CutAboveCommand(raw, "unrelated cmd"); got != raw {
		t.Errorf("CutAboveCommand changed input: %q", got)
	}
}

func TestTrimTailKeepsLastLines(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 100; i++ {
		b.WriteString("line\n")
	}
	b.WriteString("THE-ERROR\n")
	got := TrimTail(b.String(), 10, 1<<20)
	if strings.Count(got, "line\n") > 10 {
		t.Errorf("tail not trimmed: %d lines", strings.Count(got, "\n"))
	}
	if !strings.Contains(got, "THE-ERROR") {
		t.Errorf("tail lost the error line")
	}
	if !strings.Contains(got, "omitted") {
		t.Errorf("missing omission marker")
	}
}

func TestTrimTailByteCap(t *testing.T) {
	s := strings.Repeat("x", 500) + "\nEND"
	got := TrimTail(s, 1000, 100)
	if len(got) > 100+len("(earlier output omitted)\n")+1 {
		t.Errorf("byte cap not enforced: %d bytes", len(got))
	}
	if !strings.Contains(got, "END") {
		t.Errorf("byte cap kept the wrong end")
	}
}

func TestForModelEndToEnd(t *testing.T) {
	raw := "\x1b[32muser@host:~$ git push\x1b[0m\nEverything up-to-date\nuser@host:~$ "
	got := ForModel(raw, "git push")
	if got != "Everything up-to-date" {
		t.Errorf("ForModel = %q", got)
	}
	if ForModel("", "x") != "" {
		t.Errorf("empty input should stay empty")
	}
}
