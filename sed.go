package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"os/exec"
)

const SedToolName = "sed_all_in_one"

// Sed represents a wrapper around the system's sed command-line tool
type Sed struct {
	logger      observability.Logger
	cmdExecutor CommandExecutor
}

// NewSed creates a new instance of the Sed wrapper
func NewSed(logger observability.Logger) *Sed {
	return &Sed{
		logger:      logger,
		cmdExecutor: &RealCommandExecutor{},
	}
}

// SedAllInOneTool returns a mcp.Tool that can execute sed commands
// SedAllInOneTool returns a mcp.Tool that can execute sed commands
func (s *Sed) SedAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        SedToolName,
		Description: "Stream editor for filtering and transforming text",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "expression": {
                    "type": "string",
                    "description": "Sed expression/pattern to apply"
                },
                "files": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "Files to process"
                },
                "options": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "Additional sed options (e.g., -i for in-place editing)"
                }
            },
            "required": ["expression"]
        }`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			var input struct {
				Expression string   `json:"expression"`
				Files      []string `json:"files"`
				Options    []string `json:"options"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return mcp.CallToolResult{
					Content: []mcp.ToolResultContent{
						{
							Type: "text",
							Text: fmt.Sprintf("Failed to parse input: %s", err.Error()),
						},
					},
					IsError: true,
				}, nil
			}

			args := append(input.Options, input.Expression)
			if len(input.Files) > 0 {
				args = append(args, input.Files...)
			}

			s.logger.Info("Executing sed command", "expression", input.Expression, "files", input.Files, "options", input.Options)
			cmd := exec.Command("sed", args...)
			output, err := s.cmdExecutor.ExecuteCommand(ctx, cmd)

			if err != nil {
				if exitError, ok := err.(*exec.ExitError); ok {
					errorMsg := string(exitError.Stderr)
					if errorMsg == "" {
						errorMsg = err.Error()
					}

					s.logger.WithFields(map[string]interface{}{
						observability.ErrorLogField: err,
						"command":                   "sed",
						"args":                      args,
						"exit_code":                 exitError.ExitCode(),
						"stderr":                    errorMsg,
					}).Error("Sed command execution failed")

					return mcp.CallToolResult{
						Content: []mcp.ToolResultContent{
							{
								Type: "text",
								Text: fmt.Sprintf("Sed command failed (exit code %d): %s\nCommand: sed %v",
									exitError.ExitCode(), errorMsg, args),
							},
						},
						IsError: true,
					}, nil
				}

				// Handle non-exit errors
				return mcp.CallToolResult{
					Content: []mcp.ToolResultContent{
						{
							Type: "text",
							Text: fmt.Sprintf("Command execution error: %s\nCommand: sed %v",
								err.Error(), args),
						},
					},
					IsError: true,
				}, nil
			}

			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{Type: "text", Text: string(output)}},
				IsError: false,
			}, nil
		},
	}
}
