package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/maccavelli/mcplib"

	"github.com/maccavelli/mcp-server-sequential-thinking/internal/engine"
	"github.com/maccavelli/mcp-server-sequential-thinking/internal/handler/sequentialthinking"
	"github.com/maccavelli/mcp-server-sequential-thinking/internal/handler/system"
	"github.com/maccavelli/mcp-server-sequential-thinking/internal/meta"
	"github.com/maccavelli/mcp-server-sequential-thinking/internal/registry"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var exitFunc = os.Exit

// applyRuntimeDefaults installs conservative runtime guardrails when the
// standalone process is launched without externally managed limits. Unlike
// os.Setenv, the runtime APIs take effect after process init (the Go runtime
// reads GOMEMLIMIT/GOMAXPROCS only once, before main), so an externally set
// env value already won and we must not override it.
func applyRuntimeDefaults() {
	if _, exists := os.LookupEnv("GOMEMLIMIT"); !exists {
		debug.SetMemoryLimit(1024 << 20) // 1 GiB soft limit
	}
	if _, exists := os.LookupEnv("GOMAXPROCS"); !exists {
		runtime.GOMAXPROCS(2)
	}
}

func main() {
	applyRuntimeDefaults()

	var versionFlag bool
	flag.BoolVar(&versionFlag, "version", false, "Print version and exit")
	flag.Parse()

	if versionFlag {
		printVersion()
		exitFunc(0)
	}

	realStdout := os.Stdout
	os.Stdout = os.Stderr

	rootCtx := context.Background()
	ctx, stop := signal.NotifyContext(rootCtx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	buffer := mcplib.NewLogBuffer()
	cleanupLogs := mcplib.SetupStandardLogging("sequential-thinking", buffer)
	defer cleanupLogs()

	logger := slog.Default()
	slog.Info("[BACKPLANE] SPAWN "+meta.Name, "version", Version)

	pipeline := mcplib.NewStdioPipeline(os.Stdin, realStdout, stop)

	err := run(ctx, buffer, logger, pipeline.Reader, pipeline.Writer)
	if flushErr := pipeline.Flush(); flushErr != nil {
		slog.Debug("final flush error", "error", flushErr)
	}
	if err != nil {
		if mcplib.IsExpectedShutdownErr(err) {
			slog.Info("server shut down gracefully", "error", err)
			exitFunc(0)
			return
		}
		slog.Error("server fatal error", "error", err)
		exitFunc(1)
	}
}

func run(ctx context.Context, buffer *mcplib.LogBuffer, logger *slog.Logger, reader io.ReadCloser, writer io.WriteCloser) error {
	eng := engine.NewEngine(engine.ConfigFromEnv())

	sequentialthinking.Register(eng)
	system.Register(buffer, eng)

	mcpServer := mcplib.NewMCPServer(meta.Name, Version, logger)

	for _, t := range registry.Global.List() {
		t.Register(mcpServer.MCPServer())
	}

	mcpServer.MCPServer().AddResource(&mcp.Resource{
		Name:        "Logs",
		URI:         "sequential-thinking://logs",
		Description: "Server logs",
		MIMEType:    "text/plain",
	}, mcplib.HardenedResourceHandler(func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      request.Params.URI,
					Text:     buffer.String(),
					MIMEType: "text/plain",
				},
			},
		}, nil
	}))

	errChan := make(chan error, 1)
	go func(threadCtx context.Context) {
		if err := mcpServer.Serve(threadCtx, writer, reader); err != nil {
			select {
			case errChan <- err:
			case <-threadCtx.Done():
			}
		}
	}(ctx)

	select {
	case <-ctx.Done():
		slog.Info("context cancelled; initiating graceful shutdown")
	case err := <-errChan:
		if mcplib.IsExpectedShutdownErr(err) {
			slog.Info("stdio transport closed gracefully", "reason", err.Error())
			return nil
		}
		return fmt.Errorf("server error: %w", err)
	}

	// 🛡️ DRAIN: Allow up to 2 seconds for clean transport closure before process
	// exit, but keep watching errChan so a genuine serve error that races the
	// signal is still surfaced (and the wait returns early once serve exits).
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer drainCancel()
	select {
	case err := <-errChan:
		if err != nil && !mcplib.IsExpectedShutdownErr(err) {
			return fmt.Errorf("server error during drain: %w", err)
		}
	case <-drainCtx.Done():
	}
	slog.Info("shutdown drain period complete")
	return nil
}
