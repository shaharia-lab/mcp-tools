package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"go.opentelemetry.io/otel/attribute"
)

const DockerToolName = "docker_all_in_one"

// Docker represents a wrapper around the system's docker command-line tool
type Docker struct {
	logger      observability.Logger
	cmdExecutor CommandExecutor
}

// DockerConfig holds the configuration for the Docker tool
type DockerConfig struct {
	// Add configuration options as needed
}

// NewDocker creates and returns a new instance of the Docker wrapper
func NewDocker(logger observability.Logger, config DockerConfig) *Docker {
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

			startTime := time.Now()
			d.logger.WithFields(map[string]interface{}{
				"tool_name": params.Name,
				"arguments": string(params.Arguments),
				"timestamp": startTime.Format(time.RFC3339),
			}).Info("Starting docker command execution")

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

			span.SetAttributes(
				attribute.String("docker.command", input.Command),
				attribute.StringSlice("docker.args", input.Args),
			)

			args := append([]string{input.Command}, input.Args...)
			cmd := exec.CommandContext(ctx, "docker", args...)

			d.logger.WithFields(map[string]interface{}{
				"command": input.Command,
				"args":    input.Args,
			}).Info("Executing docker command")

			output, err := d.cmdExecutor.ExecuteCommand(ctx, cmd)
			executionTime := time.Since(startTime)

			if err != nil {
				d.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"output":                    string(output),
					"duration_ms":               executionTime.Milliseconds(),
				}).Error("Docker command failed")
				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("docker command failed: %w", err)
			}

			d.logger.Info("Docker command completed successfully",
				"duration_ms", executionTime.Milliseconds(),
				"output_size", len(output),
			)

			span.SetAttributes(
				attribute.Int64("duration_ms", executionTime.Milliseconds()),
				attribute.Int("response_size", len(output)),
			)

			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{
					{
						Type: "text",
						Text: string(output),
					},
				},
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
