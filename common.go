package mcptools

import "github.com/shaharia-lab/goai/mcp"

func returnErrorOutput(err error) mcp.CallToolResult {
	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{
			{
				Type: "text",
				Text: err.Error(),
			},
		},
		IsError: true,
	}
}
