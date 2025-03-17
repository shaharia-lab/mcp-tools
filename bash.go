package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"os/exec"
)

const BashToolName = "bash"

// Bash represents a wrapper around the system's bash command-line tool
type Bash struct {
	logger      observability.Logger
	cmdExecutor CommandExecutor
}

// NewBash creates a new instance of the Bash wrapper
func NewBash(logger observability.Logger) *Bash {
	return &Bash{
		logger:      logger,
		cmdExecutor: &RealCommandExecutor{},
	}
}

// BashAllInOneTool returns a mcp.Tool that can execute bash commands
func (b *Bash) BashAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        BashToolName,
		Description: "Execute bash commands with specified script or command",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "command": {
                    "type": "string",
                    "description": "Bash command or script to execute"
                },
                "args": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "Additional arguments for the command"
                }
            },
            "required": ["command"]
        }`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			var input struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return mcp.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			b.logger.Info("Executing bash command", "command", input.Command, "args", input.Args)
			cmd := exec.Command("bash", append([]string{"-c", input.Command}, input.Args...)...)
			output, err := b.cmdExecutor.ExecuteCommand(ctx, cmd)
			if err != nil {
				return mcp.CallToolResult{}, fmt.Errorf("bash command failed: %w", err)
			}

			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{Type: "text", Text: string(output)}},
				IsError: false,
			}, nil
		},
	}
}
