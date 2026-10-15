package prompt

import (
	"strings"
	"testing"

	"wtf/internal/facts"
)

func TestBuildContainsFacts(t *testing.T) {
	f := facts.Facts{
		Cmd:   "npm test",
		Exit:  1,
		OK:    false,
		Cwd:   "/home/user/app",
		Git:   "main, 2 modified",
		Trace: "FAIL src/app.test.ts",
	}
	sys, user := Build(StatusFail, f)

	for _, want := range []string{
		"command: npm test",
		"exit code: 1 (failed)",
		"cwd: /home/user/app",
		"git: main, 2 modified",
		"FAIL src/app.test.ts",
	} {
		if !strings.Contains(user, want) {
			t.Errorf("user prompt missing %q:\n%s", want, user)
		}
	}
	if !strings.Contains(sys, "ONE OR TWO sentences") {
		t.Errorf("system prompt missing failure instruction:\n%s", sys)
	}
}

func TestBuildNoTrace(t *testing.T) {
	_, user := Build(StatusOK, facts.Facts{Cmd: "ls", Exit: 0, OK: true})
	if !strings.Contains(user, "trace: not captured") {
		t.Errorf("missing no-trace marker:\n%s", user)
	}
}

func TestBuildFixInstruction(t *testing.T) {
	sys, _ := Build(Fix, facts.Facts{Cmd: "ls", Exit: 0, OK: true})
	if !strings.Contains(sys, "WTF_UNFIXABLE") || !strings.Contains(sys, "one line") {
		t.Errorf("fix instruction wrong:\n%s", sys)
	}
}
