> **Mirror notice:** This repository is a one-way published export of a
> privately hosted project. History is squashed into sync snapshots, and pull
> requests cannot be merged here directly — open an issue instead. Changes
> land in the private source and are re-exported.

<!-- markdownlint-disable MD013 MD060 MD033 -->

# MagicSequential-Thinking MCP Sub-Server

A high-performance Model Context Protocol (MCP) sub-server for structured, deep cognitive reasoning and self-correcting logic loops.

## Overview

`mcp-server-sequential-thinking` provides a profound cognitive processing lattice. It is designed to help AI agents map intricate paradoxes, unravel complex logic topologies, and perform rigorous self-critique before committing to terminal actions.

### Core Capabilities

| Feature | Description |
|---|---|
| **Iterative Reasoning** | Guides the agent through multiple steps, documenting assumptions and in-place revisions. |
| **Self-Critique** | Recommended — optional `selfCritique` field encourages self-correction; not enforced at runtime. |
| **Branching Logic** | Explore multiple reasoning paths via tracked branch IDs; conclusions remain linear append (no automatic merge). |
| **Ephemeral Session State** | In-memory per-session history with bounded retention; evicted under LRU pressure; not returned via API. |

---

## Quick Start

### Step 1: Place the Binary

Download the `mcp-server-sequential-thinking` binary for your platform and place it in a directory on your system `PATH`.

#### Linux

```bash
# Move the binary to your local bin directory
mv mcp-server-sequential-thinking ~/.local/bin/mcp-server-sequential-thinking
chmod +x ~/.local/bin/mcp-server-sequential-thinking
```

#### macOS

```bash
# Move the binary to your local bin directory
mv mcp-server-sequential-thinking /usr/local/bin/mcp-server-sequential-thinking
chmod +x /usr/local/bin/mcp-server-sequential-thinking
```

#### Windows (PowerShell)

```powershell
# Create a directory for the binary if it doesn't exist
New-Item -ItemType Directory -Force -Path "$env:LOCALAPPDATA\Programs\sequential-thinking"

# Move the binary
Move-Item mcp-server-sequential-thinking.exe "$env:LOCALAPPDATA\Programs\sequential-thinking\mcp-server-sequential-thinking.exe"

# Add to your PATH (current user, persistent)
$currentPath = [Environment]::GetEnvironmentVariable("Path", "User")
[Environment]::SetEnvironmentVariable("Path", "$currentPath;$env:LOCALAPPDATA\Programs\sequential-thinking", "User")
```

---

### Step 2: Initialize Configuration

`mcp-server-sequential-thinking` maintains **ephemeral in-memory session state** (bounded thought history and branch tracking per session). It does not require local configuration files, API tokens, or initialization steps like `init` or `configure`.

Sessions are identified by the optional `session_id` tool input, or by a per-connection fallback when omitted. History is retained only in-process and is discarded when a session is evicted from the LRU cache or the process exits. Prior thought content is **not** returned to clients — only counters and metadata in the tool response.

---

### Step 3: Configure Your IDE

> **⚠️ IMPORTANT ORCHESTRATOR MESSAGING**
>
> While the standalone IDE configurations below are provided for testing and debugging, `mcp-server-sequential-thinking` is designed to be run as a downstream node behind the **`magictools` orchestrator** in production environments.
>
> When running in production, you should **only** configure `magictools` in your IDE, which will automatically proxy requests to `sequential-thinking` as needed.

If you are testing the server standalone, configure your IDE to launch the binary directly (no `serve` argument is needed):

#### Antigravity (Google DeepMind)

| OS | Configuration File Path |
|---|---|
| Linux | `~/.gemini/antigravity/mcp_config.json` |
| macOS | `~/.gemini/antigravity/mcp_config.json` |
| Windows | `%USERPROFILE%\.gemini\antigravity\mcp_config.json` |

**Linux / macOS:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/home/youruser/.local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/home/youruser"
      }
    }
  }
}
```

**Windows:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "C:\\Users\\YourUser\\AppData\\Local\\Programs\\sequential-thinking\\mcp-server-sequential-thinking.exe",
      "env": {
        "USERPROFILE": "C:\\Users\\YourUser"
      }
    }
  }
}
```

#### Visual Studio Code (GitHub Copilot / Native MCP)

| OS | User-Level Configuration File Path |
|---|---|
| Linux | `~/.config/Code/User/mcp.json` |
| macOS | `~/Library/Application Support/Code/User/mcp.json` |
| Windows | `%APPDATA%\Code\User\mcp.json` |

**Linux:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/home/youruser/.local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/home/youruser"
      }
    }
  }
}
```

**macOS:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/usr/local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/Users/youruser"
      }
    }
  }
}
```

**Windows:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "C:\\Users\\YourUser\\AppData\\Local\\Programs\\sequential-thinking\\mcp-server-sequential-thinking.exe",
      "env": {
        "USERPROFILE": "C:\\Users\\YourUser"
      }
    }
  }
}
```

#### VSCode — Cline Extension

| OS | Configuration File Path |
|---|---|
| Linux | `~/.cline/data/settings/cline_mcp_settings.json` |
| macOS | `~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json` |
| Windows | `%APPDATA%\Code\User\globalStorage\saoudrizwan.claude-dev\settings\cline_mcp_settings.json` |

**Linux:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/home/youruser/.local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/home/youruser"
      }
    }
  }
}
```

**macOS:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/usr/local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/Users/youruser"
      }
    }
  }
}
```

**Windows:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "C:\\Users\\YourUser\\AppData\\Local\\Programs\\sequential-thinking\\mcp-server-sequential-thinking.exe",
      "env": {
        "USERPROFILE": "C:\\Users\\YourUser"
      }
    }
  }
}
```

#### Claude Desktop

| OS | Configuration File Path |
|---|---|
| macOS | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows | `%APPDATA%\Claude\claude_desktop_config.json` |

**macOS:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/usr/local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/Users/youruser"
      }
    }
  }
}
```

**Windows:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "C:\\Users\\YourUser\\AppData\\Local\\Programs\\sequential-thinking\\mcp-server-sequential-thinking.exe",
      "env": {
        "USERPROFILE": "C:\\Users\\YourUser"
      }
    }
  }
}
```

#### Claude Code (CLI)

Claude Code uses a CLI command to register MCP servers.

**Linux:**

```bash
claude mcp add sequential-thinking -s user -- /home/youruser/.local/bin/mcp-server-sequential-thinking
```

**macOS:**

```bash
claude mcp add sequential-thinking -s user -- /usr/local/bin/mcp-server-sequential-thinking
```

**Windows (PowerShell):**

```powershell
claude mcp add sequential-thinking -s user -- "C:\Users\YourUser\AppData\Local\Programs\sequential-thinking\mcp-server-sequential-thinking.exe"
```

#### Cursor

| OS | Global Configuration File Path |
|---|---|
| Linux | `~/.cursor/mcp.json` |
| macOS | `~/.cursor/mcp.json` |
| Windows | `%USERPROFILE%\.cursor\mcp.json` |

**Linux:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/home/youruser/.local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/home/youruser"
      }
    }
  }
}
```

**macOS:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/usr/local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/Users/youruser"
      }
    }
  }
}
```

**Windows:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "C:\\Users\\YourUser\\AppData\\Local\\Programs\\sequential-thinking\\mcp-server-sequential-thinking.exe",
      "env": {
        "USERPROFILE": "C:\\Users\\YourUser"
      }
    }
  }
}
```

#### JetBrains IDEs (IntelliJ, GoLand, WebStorm, PyCharm)

| OS | Global Configuration File Path |
|---|---|
| Linux | `~/.config/JetBrains/AI/mcp.json` (or via UI: Settings > Tools > AI Assistant > MCP Servers) |
| macOS | `~/Library/Application Support/JetBrains/AI/mcp.json` (or via UI: Settings > Tools > AI Assistant > MCP Servers) |
| Windows | `%APPDATA%\JetBrains\AI\mcp.json` (or via UI: Settings > Tools > AI Assistant > MCP Servers) |

**Linux:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/home/youruser/.local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/home/youruser"
      }
    }
  }
}
```

**macOS:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "/usr/local/bin/mcp-server-sequential-thinking",
      "env": {
        "HOME": "/Users/youruser"
      }
    }
  }
}
```

**Windows:**

```json
{
  "mcpServers": {
    "sequential-thinking": {
      "command": "C:\\Users\\YourUser\\AppData\\Local\\Programs\\sequential-thinking\\mcp-server-sequential-thinking.exe",
      "env": {
        "USERPROFILE": "C:\\Users\\YourUser"
      }
    }
  }
}
```

---

## Configuration

Optional environment variables tune engine limits (invalid or non-positive values are ignored):

| Variable | Default | Purpose |
|---|---|---|
| `SEQ_THINKING_LRU_SIZE` | 1000 | Distinct sessions retained in the LRU cache |
| `SEQ_THINKING_MAX_HISTORY` | 1000 | Retained thoughts per session (ring buffer) |
| `SEQ_THINKING_MAX_BRANCHES` | 64 | Distinct branch IDs per session |
| `SEQ_THINKING_MAX_BRANCH_REFS` | 128 | Thought-number references tracked per branch |
| `SEQ_THINKING_MAX_THOUGHT_BYTES` | 16384 | Maximum accepted size of a single thought body |
| `SEQ_THINKING_MAX_TOTAL_THOUGHTS` | 10000 | Upper bound for `thoughtNumber` / `totalThoughts` |
| `DISABLE_THOUGHT_LOGGING` | false | Suppress per-thought INFO logging when `true` |

---

## MCP Tools Reference

Once the server is running, the following tools are exposed to your IDE's AI agent:

| Tool | Description |
|---|---|
| `sequentialthinking` | Primary reasoning tool. Requires `thought`, `nextThoughtNeeded`, `thoughtNumber`, and `totalThoughts`. |
| `get_internal_logs` | Returns recent server log lines from the in-memory ring buffer. |
| `get_engine_stats` | Returns runtime counters: thoughts processed, handler errors, evictions, resumed-empty sessions, active session count. |

### `sequentialthinking` Output Fields

| Field | Meaning |
|---|---|
| `thoughtHistoryLength` | Lifetime count of thoughts processed in this session (monotonic, includes revisions). |
| `retainedHistoryLength` | Current size of the in-memory ring buffer (may be less than `thoughtHistoryLength` after eviction). |
| `branches` | Sorted list of active branch IDs for this session. |

### Revision

Set `isRevision: true` with `revisesThought` set to the `thoughtNumber` to replace. The engine splices the new content into the retained ring buffer in-place (no append). If the target thought was evicted from the ring, the revision is rejected.

### Continuous Dialectic Enforcement

The engine enforces two rules when `contradictionDetected` is used:

1. If the **previous retained thought** had `contradictionDetected: true`, the next thought **must** include a non-empty `resolutionStrategy`.
2. A thought cannot set `nextThoughtNeeded: false` while `contradictionDetected: true` on the same thought (the contradiction must be resolved first).

Violations return a `CONTINUOUS DIALECTIC VIOLATION` error.

---

## Data Storage Locations

| Data | Linux | macOS | Windows |
|---|---|---|---|
| Server Logs | `stderr` (captured by IDE) | `stderr` (captured by IDE) | `stderr` (captured by IDE) |

---

*Built with Go. Part of the MagicTools Intelligence Suite.*
