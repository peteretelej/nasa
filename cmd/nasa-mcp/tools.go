package main

import (
	"context"

	"github.com/peteretelej/nasa"
)

// tools is the registry of all available MCP tools. Phase 2 replaces
// the placeholder below with real tool definitions.
var tools = []ToolDef{
	{
		Tool: Tool{
			Name:        "nasa_apod",
			Description: "Get NASA Astronomy Picture of the Day",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date": map[string]any{
						"type":        "string",
						"description": "Date in YYYY-MM-DD format (default: today)",
					},
				},
			},
		},
		Handler: func(_ context.Context, _ map[string]any, _ *nasa.Client) *ToolResult {
			return &ToolResult{
				Content: []Content{
					{Type: "text", Text: "APOD placeholder - will be implemented in Phase 2"},
				},
			}
		},
	},
}
