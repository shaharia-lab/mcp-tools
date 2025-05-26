package mcptools

import "github.com/shaharia-lab/goai"

func returnErrorOutput(err error) goai.CallToolResult {
	return goai.CallToolResult{
		Content: []goai.ToolResultContent{
			{
				Type: "text",
				Text: err.Error(),
			},
		},
		IsError: true,
	}
}
