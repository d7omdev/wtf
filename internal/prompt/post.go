package prompt

import "strings"

// Post renders the model's raw answer as terse, clean terminal text.
func Post(m Mode, out string) string {
	s := tidy(out)
	switch m {
	case StatusOK, StatusFail:
		return clampSentences(s, 2, 25)
	case Explain:
		return clampSentences(s, 3, 60)
	}
	return s
}

// PostFix extracts a single runnable command line from the model's answer.
// ok=false means no fix was found.
func PostFix(out string) (string, bool) {
	for _, line := range strings.Split(extractFence(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.Trim(line, "`")
		line = strings.TrimPrefix(line, "$ ")
		line = strings.TrimPrefix(line, "> ")
		line = strings.Trim(line, "\"'")
		line = stripFixPrefixes(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.EqualFold(line, "WTF_UNFIXABLE") {
			return "", false
		}
		return line, true
	}
	return "", false
}

func stripFixPrefixes(line string) string {
	for _, p := range []string{
		"the fix is:", "the corrected command is:", "the corrected command:",
		"corrected command:", "fixed command:", "run:", "use:", "try:", "command:",
	} {
		if len(line) > len(p) && strings.EqualFold(line[:len(p)], p) {
			return strings.TrimSpace(line[len(p):])
		}
	}
	return line
}

func tidy(s string) string {
	s = extractFence(s)
	s = strings.TrimSpace(s)
	if len(s) >= 2 {
		q := s[0]
		if (q == '"' || q == '\'' || q == '`') && s[len(s)-1] == q {
			s = s[1 : len(s)-1]
		}
	}
	return strings.TrimSpace(s)
}

// extractFence pulls content out of ``` fences when present.
func extractFence(s string) string {
	if !strings.Contains(s, "```") {
		return s
	}
	var out []string
	inFence := false
	fenced := false
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "```") {
			inFence = !inFence
			fenced = true
			continue
		}
		if inFence {
			out = append(out, line)
		}
	}
	if fenced && len(out) > 0 {
		return strings.Join(out, "\n")
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "```", ""), "`", "")
}

func clampSentences(s string, maxSent, maxWords int) string {
	sents := splitSentences(s)
	if len(sents) > maxSent {
		sents = sents[:maxSent]
	}
	s = strings.Join(sents, " ")
	if words := strings.Fields(s); len(words) > maxWords {
		s = strings.Join(words[:maxWords], " ") + "..."
	}
	return strings.TrimSpace(s)
}

// splitSentences splits on ". ", "! ", "? " boundaries after collapsing
// newlines (models like to answer in lines).
func splitSentences(s string) []string {
	s = strings.Join(strings.Fields(s), " ")
	var out []string
	var cur strings.Builder
	runes := []rune(s)
	for i, r := range runes {
		cur.WriteRune(r)
		if r == '.' || r == '!' || r == '?' {
			if i+1 == len(runes) || runes[i+1] == ' ' {
				if strings.TrimSpace(string(runes[i+1:])) != "" {
					out = append(out, strings.TrimSpace(cur.String()))
					cur.Reset()
				}
			}
		}
	}
	if t := strings.TrimSpace(cur.String()); t != "" {
		out = append(out, t)
	}
	return out
}
