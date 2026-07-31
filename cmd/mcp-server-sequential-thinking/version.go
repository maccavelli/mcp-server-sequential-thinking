package main

import (
	"fmt"
	"os"
	"strings"
)

// Version is the current version of the Sequential Thinking MCP server.
var RawVersion = "v4.3.2"
var Version = strings.TrimPrefix(RawVersion, "v")

func printVersion() {
	fmt.Fprintf(os.Stderr, "mcp-server-sequential-thinking version %s\n", Version)
}
