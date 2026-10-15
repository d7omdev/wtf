package prompt

import "testing"

func TestPostStatusClampsSentences(t *testing.T) {
	in := "The build failed because Go rejects the old syntax. It printed three errors. A fourth sentence follows here."
	got := Post(StatusFail, in)
	if want := 2; countWordsOK(got, 25) != true {
		t.Errorf("status answer over word budget: %q", got)
	} else {
		_ = want
	}
	if n := sentenceCount(got); n > 2 {
		t.Errorf("status answer has %d sentences, want <= 2: %q", n, got)
	}
}

func TestPostExplainClampsThreeSentences(t *testing.T) {
	in := "One. Two. Three. Four."
	got := Post(Explain, in)
	if n := sentenceCount(got); n > 3 {
		t.Errorf("explain has %d sentences, want <= 3: %q", n, got)
	}
}

func TestPostStripsFencesAndQuotes(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"```npm ci```", "npm ci"},
		{"```\nthe command timed out\n```", "the command timed out"},
		{`"the file is missing"`, "the file is missing"},
		{"  plain answer  ", "plain answer"},
	} {
		if got := Post(StatusFail, tc.in); got != tc.want {
			t.Errorf("Post(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPostLongSentenceWordCap(t *testing.T) {
	in := "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty twentyone twentytwo twentythree twentyfour twentyfive twentysix"
	got := Post(StatusOK, in)
	if !countWordsOK(got, 25) {
		t.Errorf("word cap not enforced: %q", got)
	}
}

func TestPostFixExtractsCommand(t *testing.T) {
	for _, tc := range []struct {
		name   string
		in     string
		want   string
		wantOK bool
	}{
		{"plain", "pip install requests", "pip install requests", true},
		{"fenced", "```sh\npip install requests\n```", "pip install requests", true},
		{"prompt symbol", "$ ls -la", "ls -la", true},
		{"the fix is", "The fix is: npm ci", "npm ci", true},
		{"run prefix", "Run: git status", "git status", true},
		{"unfixable", "WTF_UNFIXABLE", "", false},
		{"unfixable fenced", "```\nWTF_UNFIXABLE\n```", "", false},
		{"empty", "   ", "", false},
		{"comment first", "# try this\nrm -f /tmp/x", "rm -f /tmp/x", true},
		{"quoted", "\"grep foo file\"", "grep foo file", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := PostFix(tc.in)
			if ok != tc.wantOK || got != tc.want {
				t.Errorf("PostFix(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

// helpers -------------------------------------------------------------------

func countWordsOK(s string, max int) bool { return wordCount(s) <= max }

func wordCount(s string) int {
	n := 0
	in := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' {
			in = false
			continue
		}
		if !in {
			n++
			in = true
		}
	}
	return n
}

func sentenceCount(s string) int {
	n := 0
	for _, r := range s {
		if r == '.' || r == '!' || r == '?' {
			n++
		}
	}
	if n == 0 && s != "" {
		return 1
	}
	return n
}
