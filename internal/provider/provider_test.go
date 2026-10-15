package provider

import "testing"

func TestGetKnownProviders(t *testing.T) {
	for _, name := range []string{"claude", "codex", "opencode", "ollama", "mock"} {
		p, err := Get(name)
		if err != nil {
			t.Fatalf("Get(%q): %v", name, err)
		}
		if p.Name() != name {
			t.Errorf("Name() = %q, want %q", p.Name(), name)
		}
	}
}

func TestGetUnknown(t *testing.T) {
	if _, err := Get("nope"); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestOllamaBaseHonorsEnv(t *testing.T) {
	for env, want := range map[string]string{
		"":                       "http://127.0.0.1:11434",
		"localhost:11435":        "http://localhost:11435",
		"http://10.0.0.5:11434/": "http://10.0.0.5:11434",
	} {
		t.Setenv("OLLAMA_HOST", env)
		if got := ollamaBase(); got != want {
			t.Errorf("OLLAMA_HOST=%q → %q, want %q", env, got, want)
		}
	}
}

func TestAllowReadonlyPermission(t *testing.T) {
	for _, tc := range []struct {
		params string
		want   bool
	}{
		{`{"toolCalls":[{"kind":"read","title":"x"}]}`, true},
		{`{"toolCalls":[{"kind":"execute","title":"run"}]}`, false},
		{`{"toolCalls":[{"kind":"edit","title":"write"}]}`, false},
		{`{"toolCalls":[{"tool":"grep","title":"search"}]}`, true},
		{`{}`, false},
		{`garbage`, false},
	} {
		if got := allowReadonlyPermission([]byte(tc.params)); got != tc.want {
			t.Errorf("allowReadonlyPermission(%s) = %v, want %v", tc.params, got, tc.want)
		}
	}
}
