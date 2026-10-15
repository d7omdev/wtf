package provider

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestMain doubles as the fake `opencode acp` agent: when WTF_ACP_HELPER=1 the
// test binary speaks just enough ACP to exercise the client end-to-end.
func TestMain(m *testing.M) {
	if os.Getenv("WTF_ACP_HELPER") == "1" {
		runFakeAgent()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func runFakeAgent() {
	in := bufio.NewScanner(os.Stdin)
	in.Buffer(make([]byte, 0, 1<<16), 1<<20)
	out := os.Stdout
	send := func(v any) {
		b, _ := json.Marshal(v)
		fmt.Fprintln(out, string(b))
	}
	permissionOK := false

	for in.Scan() {
		var req struct {
			ID     *int64          `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(in.Bytes(), &req); err != nil || req.Method == "" {
			continue
		}
		id := *req.ID
		switch req.Method {
		case "initialize":
			send(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
				"protocolVersion": 1,
				"agentInfo":       map[string]any{"name": "fake-agent", "version": "1.0"},
			}})
		case "session/new":
			send(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"sessionId": "sess_1"}})
		case "session/set_config_option":
			send(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{}})
		case "session/prompt":
			// reverse request: ask permission for a read-only tool
			send(map[string]any{"jsonrpc": "2.0", "id": 9999, "method": "session/request_permission",
				"params": map[string]any{
					"sessionId": "sess_1",
					"toolCalls": []map[string]any{{"title": "read file", "kind": "read"}},
				}})
			// read the client's reply before answering the prompt
			if in.Scan() {
				var reply struct {
					ID     *int64 `json:"id"`
					Result struct {
						Response string `json:"response"`
					} `json:"result"`
				}
				if json.Unmarshal(in.Bytes(), &reply) == nil && reply.Result.Response == "allow_once" {
					permissionOK = true
				}
			}
			// stream two message chunks, then answer the prompt
			for _, chunk := range []string{"hello ", "from agent"} {
				send(map[string]any{"jsonrpc": "2.0", "method": "session/update",
					"params": map[string]any{"update": map[string]any{
						"sessionUpdate": "agent_message_chunk",
						"content":       map[string]any{"type": "text", "text": chunk},
					}}})
			}
			if !permissionOK {
				send(map[string]any{"jsonrpc": "2.0", "method": "session/update",
					"params": map[string]any{"update": map[string]any{
						"sessionUpdate": "agent_message_chunk",
						"content":       map[string]any{"type": "text", "text": " | PERMISSION_PROTOCOL_ERROR"},
					}}})
			}
			send(map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"stopReason": "end_turn"}})
		default:
			send(map[string]any{"jsonrpc": "2.0", "id": id,
				"error": map[string]any{"code": -32601, "message": "unknown method"}})
		}
	}
}

func TestOpencodeACPComplete(t *testing.T) {
	// a stand-in `opencode` binary that re-execs this test as the fake agent
	dir := t.TempDir()
	script := fmt.Sprintf("#!/bin/sh\nexport WTF_ACP_HELPER=1\nexec '%s' acp\n", os.Args[0])
	bin := dir + "/opencode"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	p := opencodeProvider{}
	out, err := p.Complete(t.Context(), Request{
		System: "system prompt",
		Prompt: "user prompt",
		Cwd:    dir,
	})
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if out != "hello from agent" {
		t.Errorf("out = %q (permission handshake worked = %v)", out, strings.Contains(out, "hello from agent") && !strings.Contains(out, "PERMISSION"))
	}
	if strings.Contains(out, "PERMISSION_PROTOCOL_ERROR") {
		t.Errorf("client's permission reply did not parse: %q", out)
	}
}
