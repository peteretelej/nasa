# MCP Server Setup

The NASA MCP server exposes NASA APIs as tools for AI assistants over the Model Context Protocol (stdio JSON-RPC 2.0).

## Installation

```bash
go install github.com/peteretelej/nasa/cmd/nasa-mcp@latest
```

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

### Claude Code

```bash
claude mcp add nasa nasa-mcp -e NASA_API_KEY=your-key
```

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

## How It Works

The MCP server communicates over stdin/stdout using JSON-RPC 2.0. Each line of stdin is a JSON-RPC request; responses are written to stdout. Diagnostic messages go to stderr.

The server exposes 13 tools covering APOD, NEO, DONKI, Images, SSD, and EONET. See [Tools](tools.md) for the full list.
