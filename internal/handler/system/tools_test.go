package system

import (
	"testing"

	"github.com/maccavelli/mcplib"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/mcp-server-sequential-thinking/internal/engine"
)

func TestRegister(t *testing.T) {
	buffer := mcplib.NewLogBuffer()
	eng := engine.NewEngine(engine.DefaultConfig())
	Register(buffer, eng)

	shim := &diagnosticToolShim{buffer: buffer}
	if shim.Name() != "get_internal_logs" {
		t.Errorf("expected get_internal_logs, got %s", shim.Name())
	}

	stats := &statsToolShim{engine: eng}
	if stats.Name() != "get_engine_stats" {
		t.Errorf("expected get_engine_stats, got %s", stats.Name())
	}

	s := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, &mcp.ServerOptions{})
	shim.Register(s)
	stats.Register(s)
}
