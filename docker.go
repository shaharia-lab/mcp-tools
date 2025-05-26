package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/shaharia-lab/goai"
)

const DockerToolName = "docker"

// Docker represents a wrapper around the system's docker command-line tool
type Docker struct {
	logger      goai.Logger
	cmdExecutor CommandExecutor
}

// Rest of your existing struct definitions...

// NewDocker creates and returns a new instance of the Docker wrapper
func NewDocker(logger goai.Logger) *Docker {
	return &Docker{
		logger:      logger,
		cmdExecutor: &RealCommandExecutor{},
	}
}

// DockerAllInOneTool returns a goai.Tool that can execute Docker commands
func (d *Docker) DockerAllInOneTool() goai.Tool {
	return goai.Tool{
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
		Handler: func(ctx context.Context, params goai.CallToolParams) (goai.CallToolResult, error) {
			ctx, span := goai.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
			defer span.End()

			var input struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			}

			d.logger.WithFields(map[string]interface{}{
				"tool": DockerToolName,
			}).Info("Received input", "input", string(params.Arguments))

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				d.logger.WithFields(map[string]interface{}{
					goai.ErrorLogField: err,
					"raw_input":                 string(params.Arguments),
				}).Error("Failed to unmarshal input parameters")
				span.RecordError(err)
				return goai.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			if err := validateDockerInput(input); err != nil {
				d.logger.WithFields(map[string]interface{}{
					goai.ErrorLogField: err,
				}).Error("Input validation failed")
				span.RecordError(err)
				return returnErrorOutput(err), nil
			}

			// Create the command with plain text output format
			args := append([]string{input.Command}, input.Args...)
			cmd := exec.Command("docker", args...)

			d.logger.WithFields(map[string]interface{}{
				"tool": DockerToolName,
			}).Info("Executing docker command", "command", input.Command, "args", input.Args)

			// Execute the command using the executor
			output, err := d.cmdExecutor.ExecuteCommand(ctx, cmd)
			if err != nil {
				d.logger.WithFields(map[string]interface{}{
					goai.ErrorLogField: err,
					"command":                   "docker",
					"args":                      args,
				}).Error("Docker command execution failed")
				span.RecordError(err)
				return returnErrorOutput(err), nil
			}

			d.logger.WithFields(map[string]interface{}{
				"tool": DockerToolName,
			}).Info("Docker command executed successfully", "command", input.Command, "args", input.Args)

			return goai.CallToolResult{
				Content: []goai.ToolResultContent{
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
