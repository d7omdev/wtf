package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// acpClientVersion is reported in the ACP initialize handshake.
const acpClientVersion = "0.1.0"

type opencodeProvider struct{}

func (opencodeProvider) Name() string { return "opencode" }

func (opencodeProvider) Complete(ctx context.Context, req Request) (string, error) {
	bin := look("opencode")
	if bin == "" {
		return "", fmt.Errorf("opencode CLI not found on PATH")
	}
	c, err := newACP(bin, req.Cwd)
	if err != nil {
		return "", err
	}
	defer c.Close()

	_, err = c.call(ctx, "initialize", map[string]any{
		"protocolVersion": 1,
		"clientCapabilities": map[string]any{
			"fs":       map[string]any{"readTextFile": true, "writeTextFile": false},
			"terminal": false,
		},
		"clientInfo": map[string]any{"name": "wtf", "version": acpClientVersion},
	}, nil)
	if err != nil {
		return "", fmt.Errorf("opencode acp initialize: %w%s", err, tail(c.stderr))
	}

	newRes, err := c.call(ctx, "session/new", map[string]any{
		"cwd":        req.Cwd,
		"mcpServers": []any{},
	}, nil)
	if err != nil {
		return "", fmt.Errorf("opencode acp session/new: %w%s", err, tail(c.stderr))
	}
	var sn struct {
		SessionID string `json:"sessionId"`
	}
	if err := json.Unmarshal(newRes, &sn); err != nil || sn.SessionID == "" {
		return "", fmt.Errorf("opencode acp: no sessionId in response%s", tail(c.stderr))
	}

	// best-effort model selection when the agent advertises config options
	if req.Model != "" {
		_, _ = c.call(ctx, "session/set_config_option", map[string]any{
			"sessionId": sn.SessionID,
			"configId":  "models",
			"value":     req.Model,
		}, nil)
	}

	var answer strings.Builder
	_, err = c.call(ctx, "session/prompt", map[string]any{
		"sessionId": sn.SessionID,
		"prompt":    []map[string]any{{"type": "text", "text": req.System + "\n\n" + req.Prompt}},
	}, func(params json.RawMessage) {
		var up struct {
			Update struct {
				SessionUpdate string `json:"sessionUpdate"`
				Content       struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"update"`
		}
		if json.Unmarshal(params, &up) == nil && up.Update.SessionUpdate == "agent_message_chunk" {
			answer.WriteString(up.Update.Content.Text)
		}
	})
	if err != nil {
		return "", fmt.Errorf("opencode acp: %w%s", err, tail(c.stderr))
	}
	out := strings.TrimSpace(answer.String())
	if out == "" {
		return "", fmt.Errorf("opencode returned an empty response%s", tail(c.stderr))
	}
	return out, nil
}

func (p opencodeProvider) Models(ctx context.Context) ([]string, error) {
	bin := look("opencode")
	if bin == "" {
		return nil, fmt.Errorf("opencode CLI not found on PATH")
	}
	cctx, cancel := withTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, bin, "models")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("opencode models: %w", err)
	}
	var models []string
	seen := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		models = append(models, line)
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("opencode: no models reported")
	}
	return models, nil
}

// --- minimal ACP client (JSON-RPC 2.0, newline-delimited, over stdio) -------

type acpConn struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
	nextID  int
	stderr  *bytes.Buffer
}

func newACP(bin, cwd string) (*acpConn, error) {
	cmd := exec.Command(bin, "acp")
	cmd.Dir = cwd
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 1<<20), 8<<20)
	return &acpConn{cmd: cmd, stdin: stdin, scanner: sc, stderr: stderr}, nil
}

type acpMsg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int64          `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *acpError       `json:"error,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type acpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// call sends a request and blocks until its response, servicing session/update
// notifications (via onUpdate) and reverse requests from the agent in between.
func (c *acpConn) call(ctx context.Context, method string, params any, onUpdate func(json.RawMessage)) (json.RawMessage, error) {
	c.nextID++
	id := c.nextID
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method}
	if params != nil {
		req["params"] = params
	}
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if _, err := c.stdin.Write(append(b, '\n')); err != nil {
		return nil, err
	}

	for {
		line, err := c.nextLine(ctx)
		if err != nil {
			return nil, err
		}
		var m acpMsg
		if err := json.Unmarshal(line, &m); err != nil {
			continue // ignore non-JSON noise
		}
		switch {
		case m.ID != nil && int64(id) == *m.ID:
			if m.Error != nil {
				return nil, fmt.Errorf("%s failed: %s (code %d)", method, m.Error.Message, m.Error.Code)
			}
			return m.Result, nil
		case m.Method == "session/update":
			if onUpdate != nil && m.Params != nil {
				onUpdate(m.Params)
			}
		case m.Method != "" && m.ID != nil:
			c.answerReverse(m)
		}
	}
}

// answerReverse replies to agent→client requests: allow clearly read-only
// tool permission asks, refuse everything else (wtf never modifies anything).
func (c *acpConn) answerReverse(m acpMsg) {
	resp := map[string]any{"jsonrpc": "2.0", "id": *m.ID}
	if m.Method == "session/request_permission" {
		if allowReadonlyPermission(m.Params) {
			resp["result"] = map[string]any{"response": "allow_once"}
		} else {
			resp["result"] = map[string]any{"response": "reject_once"}
		}
	} else {
		resp["error"] = map[string]any{"code": -32601, "message": "wtf does not support " + m.Method}
	}
	b, _ := json.Marshal(resp)
	_, _ = c.stdin.Write(append(b, '\n'))
}

func allowReadonlyPermission(params json.RawMessage) bool {
	var p struct {
		ToolCalls []struct {
			Kind string `json:"kind"`
			Tool string `json:"tool"`
		} `json:"toolCalls"`
	}
	if err := json.Unmarshal(params, &p); err != nil || len(p.ToolCalls) == 0 {
		return false
	}
	for _, tc := range p.ToolCalls {
		kind := strings.ToLower(tc.Kind)
		tool := strings.ToLower(tc.Tool)
		readonly := false
		for _, ok := range []string{"read", "search", "think", "grep", "glob", "list"} {
			if kind == ok || strings.Contains(tool, ok) {
				readonly = true
				break
			}
		}
		if !readonly {
			return false
		}
	}
	return true
}

func (c *acpConn) nextLine(ctx context.Context) ([]byte, error) {
	type res struct {
		line []byte
		err  error
	}
	ch := make(chan res, 1)
	go func() {
		if c.scanner.Scan() {
			ch <- res{c.scanner.Bytes(), nil}
		} else {
			err := c.scanner.Err()
			if err == nil {
				err = io.EOF
			}
			ch <- res{nil, err}
		}
	}()
	select {
	case r := <-ch:
		if r.err != nil {
			return nil, fmt.Errorf("connection closed: %w", r.err)
		}
		return r.line, nil
	case <-ctx.Done():
		c.Kill()
		return nil, ctx.Err()
	}
}

// Close shuts the agent down politely, then by force.
func (c *acpConn) Close() {
	_ = c.stdin.Close()
	done := make(chan struct{})
	go func() {
		_ = c.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		_ = c.cmd.Process.Kill()
	}
}

// Kill immediately terminates the agent (used on context timeout).
func (c *acpConn) Kill() {
	_ = c.cmd.Process.Kill()
}
