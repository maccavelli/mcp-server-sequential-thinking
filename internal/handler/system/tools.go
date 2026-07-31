package system

import (
	"context"

	"github.com/maccavelli/mcplib"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/mcp-server-sequential-thinking/internal/engine"
	"github.com/maccavelli/mcp-server-sequential-thinking/internal/registry"
)

// Register adds the system tools (log tailing + engine stats) to the registry.
func Register(buffer *mcplib.LogBuffer, eng *engine.Engine) {
	registry.Global.Register(&diagnosticToolShim{buffer: buffer})
	registry.Global.Register(&statsToolShim{engine: eng})
}

type diagnosticToolShim struct {
	buffer *mcplib.LogBuffer
}

func (t *diagnosticToolShim) Name() string { return "get_internal_logs" }

func (t *diagnosticToolShim) Register(s *mcp.Server) {
	mcplib.RegisterDiagnosticTool(s, t.buffer, "sequential thinking reasoning engine")
}

type statsToolShim struct {
	engine *engine.Engine
}

func (t *statsToolShim) Name() string { return "get_engine_stats" }

func (t *statsToolShim) Register(s *mcp.Server) {
	mcplib.HardenedAddSimpleTool(s, &mcp.Tool{
		Name:        t.Name(),
		Description: "Returns runtime counters for the sequential thinking engine: thoughts processed, handler errors, session evictions, resumed-empty sessions, and active session count.",
	}, func(_ context.Context, _ struct{}) (engine.Stats, error) {
		return t.engine.Stats(), nil
	})
}
