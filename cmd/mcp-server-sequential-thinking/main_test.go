package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"testing"

	"github.com/maccavelli/mcplib"
)

// TestApplyRuntimeDefaults verifies the soft memory limit is installed when
// GOMEMLIMIT is unset, and that an externally set value is left untouched.
func TestApplyRuntimeDefaults(t *testing.T) {
	// debug.SetMemoryLimit(-1) returns the current limit without changing it.
	original := debug.SetMemoryLimit(-1)
	t.Cleanup(func() { debug.SetMemoryLimit(original) })

	t.Run("unset installs 1 GiB", func(t *testing.T) {
		os.Unsetenv("GOMEMLIMIT")
		debug.SetMemoryLimit(2 << 30) // pretend a different prior value
		applyRuntimeDefaults()
		if got := debug.SetMemoryLimit(-1); got != 1<<30 {
			t.Errorf("GOMEMLIMIT default: got %d, want %d", got, 1<<30)
		}
	})

	t.Run("external value preserved", func(t *testing.T) {
		t.Setenv("GOMEMLIMIT", "512MiB")
		debug.SetMemoryLimit(512 << 20)
		applyRuntimeDefaults()
		if got := debug.SetMemoryLimit(-1); got != 512<<20 {
			t.Errorf("externally set GOMEMLIMIT should be preserved, got %d", got)
		}
	})
}

// TestIsExpectedShutdownErr verifies that the function correctly identifies expected networking errors.
func TestIsExpectedShutdownErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"EOF", io.EOF, true},
		{"unexpected EOF", io.ErrUnexpectedEOF, true},
		{"broken pipe", fmt.Errorf("broken pipe"), true},
		{"connection reset", fmt.Errorf("connection reset by peer"), true},
		{"use of closed", fmt.Errorf("use of closed network connection"), true},
		{"bad file descriptor", fmt.Errorf("bad file descriptor"), true},
		{"client is closing", fmt.Errorf("client is closing"), true},
		{"connection closed", fmt.Errorf("connection closed"), true},
		{"file already closed", fmt.Errorf("file already closed"), true},
		{"random error", fmt.Errorf("random error"), false},
		{"wrapped EOF", fmt.Errorf("wrapped: %w", io.EOF), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mcplib.IsExpectedShutdownErr(tc.err)
			if got != tc.want {
				t.Errorf("IsExpectedShutdownErr(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

// TestMain_Version verifies that the -version flag correctly outputs the version and exits.
func TestMain_Version(t *testing.T) {
	os.Args = []string{"cmd", "-version"}
	exited := false
	exitFunc = func(_ int) {
		exited = true
	}
	main()
	if !exited {
		t.Error("main should have exited")
	}
}

// TestRun_GracefulShutdown verifies that run exits gracefully when context is canceled.
func TestRun_GracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	buffer := mcplib.NewLogBuffer()
	logger := slog.Default()

	rReader, rWriter := io.Pipe()
	wReader, wWriter := io.Pipe()

	// Clean up pipes
	defer rReader.Close()
	defer rWriter.Close()
	defer wReader.Close()
	defer wWriter.Close()

	errChan := make(chan error, 1)
	go func() {
		errChan <- run(ctx, buffer, logger, rReader, wWriter)
	}()

	// Give the server a moment to start
	// We just cancel context to trigger graceful shutdown
	cancel()

	// Wait for run to return
	err := <-errChan
	if err != nil && !mcplib.IsExpectedShutdownErr(err) {
		t.Errorf("expected graceful shutdown err or nil, got: %v", err)
	}
}
