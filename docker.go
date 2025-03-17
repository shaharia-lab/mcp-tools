package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"os/exec"
)

const DockerToolName = "docker"

// Docker represents a wrapper around the system's docker command-line tool
type Docker struct {
	logger      observability.Logger
	cmdExecutor CommandExecutor
}

// Rest of your existing struct definitions...

// NewDocker creates and returns a new instance of the Docker wrapper
func NewDocker(logger observability.Logger) *Docker {
	return &Docker{
		logger:      logger,
		cmdExecutor: &RealCommandExecutor{},
	}
}

// DockerAllInOneTool returns a mcp.Tool that can execute Docker commands
func (d *Docker) DockerAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        DockerToolName,
		Description: "Execute Docker commands with specified arguments",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "command": {
                    "type": "string",
                    "description": "Docker command to execute (e.g., ps, images, run)"
                },
                "args": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "Arguments for the Docker command"
                }
            },
            "required": ["command"]
        }`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
			defer span.End()

			var input struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				d.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"raw_input":                 string(params.Arguments),
				}).Error("Failed to unmarshal input parameters")
				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			if err := validateDockerInput(input); err != nil {
				d.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
				}).Error("Input validation failed")
				span.RecordError(err)
				return mcp.CallToolResult{}, err
			}

			// Create the command with plain text output format
			args := append([]string{input.Command}, input.Args...)
			cmd := exec.Command("docker", args...)

			// Execute the command using the executor
			output, err := d.cmdExecutor.ExecuteCommand(ctx, cmd)
			if err != nil {
				d.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"command":                   "docker",
					"args":                      args,
				}).Error("Docker command execution failed")
				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("docker command failed: %w", err)
			}

			// Return plain text output
			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{
					{
						Type: "text",
						Text: string(output),
					},
				},
				IsError: false,
			}, nil
		},
	}
}

func validateDockerInput(input struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}) error {
	if input.Command == "" {
		return fmt.Errorf("command is required")
	}
	return nil
}
