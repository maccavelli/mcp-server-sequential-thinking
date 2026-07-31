package sequentialthinking

import (
	"context"
	"strings"
	"testing"

	"github.com/maccavelli/mcp-server-sequential-thinking/internal/engine"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRegister(t *testing.T) {
	eng := engine.NewEngine(engine.DefaultConfig())
	Register(eng)
	tool := &SequentialThinkingTool{Engine: eng}
	if tool.Name() != "sequentialthinking" {
		t.Error("bad name")
	}

	srv := mcp.NewServer(
		&mcp.Implementation{Name: "test", Version: "1.0"},
		&mcp.ServerOptions{},
	)
	tool.Register(srv)

	var input engine.ThoughtData
	req := &mcp.CallToolRequest{}

	// Test success
	input.Thought = "hello"
	input.ThoughtNumber = 1
	input.TotalThoughts = 1

	res, _, err := tool.Handle(context.Background(), req, input)
	if err != nil || res == nil {
		t.Error("Handle failed")
	}

	// Invalid input surfaces as an error result and increments the error metric.
	bad := engine.ThoughtData{Thought: "x", ThoughtNumber: 0, TotalThoughts: 1}
	res, _, err = tool.Handle(context.Background(), req, bad)
	if err != nil {
		t.Errorf("Handle should not return a protocol error, got %v", err)
	}
	if res == nil || !res.IsError {
		t.Error("invalid input should yield an error result")
	}
	if eng.Stats().HandlerErrors != 1 {
		t.Errorf("HandlerErrors: got %d, want 1", eng.Stats().HandlerErrors)
	}
}

func TestConnectionKey(t *testing.T) {
	if k := connectionKey(nil); k != "" {
		t.Errorf("nil req should yield empty key, got %q", k)
	}
	if k := connectionKey(&mcp.CallToolRequest{}); k != "" {
		t.Errorf("nil session should yield empty key, got %q", k)
	}

	sess := &mcp.ServerSession{}
	req := &mcp.CallToolRequest{Session: sess}
	k := connectionKey(req)
	if id := sess.ID(); id != "" {
		if k != id {
			t.Errorf("session ID: got %q, want %q", k, id)
		}
	} else if !strings.HasPrefix(k, "conn-") {
		t.Errorf("empty session ID should fall back to conn pointer key, got %q", k)
	}
}
