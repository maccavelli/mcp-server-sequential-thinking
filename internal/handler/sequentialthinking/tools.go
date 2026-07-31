package sequentialthinking

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/maccavelli/mcplib"

	"github.com/maccavelli/mcp-server-sequential-thinking/internal/engine"
	"github.com/maccavelli/mcp-server-sequential-thinking/internal/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SequentialThinkingTool wraps the Sequential Thinking Engine as an MCP tool.
type SequentialThinkingTool struct {
	Engine *engine.Engine
}

// Name returns the MCP tool identifier.
func (t *SequentialThinkingTool) Name() string {
	return "sequentialthinking"
}

// Register exposes the tool metadata and registers the handler.
func (t *SequentialThinkingTool) Register(s *mcp.Server) {
	mcplib.HardenedAddTool(s, &mcp.Tool{
		Name:        t.Name(),
		Description: `[DIRECTIVE: Deep Cognitive Processing] A profound cognitive processing lattice designed to map intricate paradoxes and abstract reasoning grids. Instead of merely executing, this engine dwells within the architecture, intentionally neutralizing cognitive drift by enforcing rigorous, expansive self-reflection over isolated neural logic loops. Engage this hyper-thread when you need to gaze deeply into a structural problem, unravel complex logic topologies, or truly ruminate on high-level architectural conceptualization before committing to terminal edits... [MANDATORY SCHEMA: thought, nextThoughtNeeded, thoughtNumber, totalThoughts] Keywords: analyze, evaluate, sequential, reason, think, consider, reflect, deliberate, ponder, contemplate, ruminate, cognitive, conceptualize, logic`,
	}, t.Handle)
}

// Handle processes incoming requests and executes thought generation.
func (t *SequentialThinkingTool) Handle(ctx context.Context, req *mcp.CallToolRequest, input engine.ThoughtData) (*mcp.CallToolResult, any, error) {
	resp, err := t.Engine.ProcessThought(ctx, connectionKey(req), input)
	if err != nil {
		t.Engine.RecordHandlerError()
		res := &mcp.CallToolResult{}
		slog.Error("handler error", "error", err)
		res.SetError(err)
		return res, nil, nil
	}
	return &mcp.CallToolResult{}, resp, nil
}

// connectionKey derives a stable per-connection identity used as the default
// session bucket when the caller omits an explicit SessionID. Streamable HTTP
// sessions carry a real ID; stdio sessions report "", so we fall back to the
// stable ServerSession pointer. This prevents distinct connections from
// sharing one process-global bucket. Returns "" when no session is available
// (e.g. in unit tests), in which case the engine uses "default_global".
func connectionKey(req *mcp.CallToolRequest) string {
	if req == nil || req.Session == nil {
		return ""
	}
	if id := req.Session.ID(); id != "" {
		return id
	}
	return fmt.Sprintf("conn-%p", req.Session)
}

// Register adds the sequential thinking tool to the global registry.
func Register(eng *engine.Engine) {
	registry.Global.Register(&SequentialThinkingTool{Engine: eng})
}
