package mcptools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
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

			c.logger.WithFields(map[string]interface{}{"tool": CatToolName}).Info("Received input", "input", string(params.Arguments))
			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return mcp.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			if len(input.Files) == 0 {
				c.logger.WithFields(map[string]interface{}{"tool": CatToolName}).Error("At least one file must be specified")
				return returnErrorOutput(errors.New("at least one file must be specified")), nil
			}

			c.logger.WithFields(map[string]interface{}{"tool": CatToolName}).Info("Total files to read", "total_files", len(input.Files))

			args := append(input.Options, input.Files...)

			c.logger.WithFields(map[string]interface{}{"tool": CatToolName}).Info("Executing cat command", "files", input.Files, "options", input.Options)
			cmd := exec.Command("cat", args...)
			output, err := c.cmdExecutor.ExecuteCommand(ctx, cmd)
			if err != nil {
				c.logger.WithFields(map[string]interface{}{"tool": CatToolName}).Error("Failed to execute cat command", "error", err)
				return returnErrorOutput(err), nil
			}

			o := string(output)
			c.logger.WithFields(map[string]interface{}{"tool": CatToolName, "output_length": len(o)}).Info("Successfully executed cat command")
			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{Type: "text", Text: o}},
				IsError: false,
			}, nil
		},
	}
}
