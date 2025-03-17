package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"os/exec"
)

const CatToolName = "cat"

// Cat represents a wrapper around the system's cat command-line tool
type Cat struct {
	logger      observability.Logger
	cmdExecutor CommandExecutor
}

// NewCat creates a new instance of the Cat wrapper
func NewCat(logger observability.Logger) *Cat {
	return &Cat{
		logger:      logger,
		cmdExecutor: &RealCommandExecutor{},
	}
}

// CatAllInOneTool returns a mcp.Tool that can execute cat commands
func (c *Cat) CatAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        CatToolName,
		Description: "Display contents of files",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "files": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "List of files to read"
                },
                "options": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "Additional cat options (e.g., -n for line numbers)"
                }
            },
            "required": ["files"]
        }`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			var input struct {
				Files   []string `json:"files"`
				Options []string `json:"options"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return mcp.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			if len(input.Files) == 0 {
				return mcp.CallToolResult{}, fmt.Errorf("at least one file must be specified")
			}

			args := append(input.Options, input.Files...)

			c.logger.Info("Executing cat command", "files", input.Files, "options", input.Options)
			cmd := exec.Command("cat", args...)
			output, err := c.cmdExecutor.ExecuteCommand(ctx, cmd)
			if err != nil {
				return mcp.CallToolResult{}, fmt.Errorf("cat command failed: %w", err)
			}

			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{Type: "text", Text: string(output)}},
				IsError: false,
			}, nil
		},
	}
}
