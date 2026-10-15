package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type ollamaProvider struct{}

func (ollamaProvider) Name() string { return "ollama" }

func ollamaBase() string {
	h := os.Getenv("OLLAMA_HOST")
	if h == "" {
		h = "127.0.0.1:11434"
	}
	if !strings.HasPrefix(h, "http://") && !strings.HasPrefix(h, "https://") {
		h = "http://" + h
	}
	return strings.TrimRight(h, "/")
}

type ollamaTags struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func (ollamaProvider) list(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaBase()+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tags ollamaTags
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	var out []string
	for _, m := range tags.Models {
		out = append(out, m.Name)
	}
	return out, nil
}

func (p ollamaProvider) count(ctx context.Context) (int, error) {
	cctx, cancel := withTimeout(ctx, time.Second)
	defer cancel()
	m, err := p.list(cctx)
	return len(m), err
}

func (p ollamaProvider) Models(ctx context.Context) ([]string, error) {
	cctx, cancel := withTimeout(ctx, 10*time.Second)
	defer cancel()
	return p.list(cctx)
}

type ollamaChatReq struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  map[string]any  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaChatResp struct {
	Message ollamaMessage `json:"message"`
	Error   string        `json:"error"`
}

func (p ollamaProvider) Complete(ctx context.Context, req Request) (string, error) {
	model := req.Model
	if model == "" {
		models, err := p.list(ctx)
		if err != nil || len(models) == 0 {
			return "", fmt.Errorf("ollama: no model configured and none available (%v)", err)
		}
		model = models[0]
	}

	body, err := json.Marshal(ollamaChatReq{
		Model:  model,
		Stream: false,
		Messages: []ollamaMessage{
			{Role: "system", Content: req.System},
			{Role: "user", Content: req.Prompt},
		},
		Options: map[string]any{"temperature": 0.2},
	})
	if err != nil {
		return "", err
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodPost, ollamaBase()+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	hreq.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return "", fmt.Errorf("ollama: %w (is `ollama serve` running?)", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var cr ollamaChatResp
	_ = json.Unmarshal(b, &cr)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: HTTP %d: %s", resp.StatusCode, cr.Error)
	}
	out := strings.TrimSpace(cr.Message.Content)
	if out == "" {
		return "", fmt.Errorf("ollama returned an empty response")
	}
	return out, nil
}
