# MCP Server Setup

The NASA MCP server exposes NASA APIs as tools for AI assistants over the Model Context Protocol (stdio JSON-RPC 2.0).

## Installation

```bash
go install github.com/peteretelej/nasa/cmd/nasa-mcp@latest
```

> **Tip:** GUI clients (Claude Desktop, Cursor, VS Code) often don't inherit your shell's `$PATH`. If the MCP server isn't found, use the full path to the binary: `$(go env GOPATH)/bin/nasa-mcp`. You can find the absolute path by running `which nasa-mcp` or `go env GOPATH` in your terminal.

## Configuration

Set your NASA API key in the `env` block. Without it, the server falls back to `DEMO_KEY` (30 requests/hour).

### Claude Desktop

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "nasa": {
      "command": "nasa-mcp",
      "env": {
        "NASA_API_KEY": "your-key"
      }
    }
  }
}
```

If `nasa-mcp` is not found, replace `"command": "nasa-mcp"` with the absolute path (e.g., `"/home/you/go/bin/nasa-mcp"`).

### Claude Code

```bash
claude mcp add nasa nasa-mcp -e NASA_API_KEY=your-key
```

If `nasa-mcp` is not in your `$PATH`, use the full path: `claude mcp add nasa "$(go env GOPATH)/bin/nasa-mcp" -e NASA_API_KEY=your-key`.

### VS Code (GitHub Copilot)

Add to `.vscode/settings.json`:

```json
{
  "github.copilot.chat.mcpServers": {
    "nasa": {
      "command": "nasa-mcp",
      "env": {
        "NASA_API_KEY": "your-key"
      }
    }
  }
}
```

If `nasa-mcp` is not found, replace `"command": "nasa-mcp"` with the absolute path.

### Cursor

Add to `.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "nasa": {
      "command": "nasa-mcp",
      "env": {
        "NASA_API_KEY": "your-key"
      }
    }
  }
}
```

If `nasa-mcp` is not found, replace `"command": "nasa-mcp"` with the absolute path.

## How It Works

The MCP server communicates over stdin/stdout using JSON-RPC 2.0. Each line of stdin is a JSON-RPC request; responses are written to stdout. Diagnostic messages go to stderr.

The server exposes 13 tools covering APOD, NEO, DONKI, Images, SSD, and EONET. See [Tools](tools.md) for the full list.
