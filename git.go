package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"go.opentelemetry.io/otel/attribute"
)

const GitToolName = "git_all_in_one"

// Git represents a wrapper around the system's git command-line tool,
// providing a programmatic interface for executing git commands.
type Git struct {
	logger observability.Logger
	config GitConfig
}

// GitConfig holds the configuration for the Git tool
type GitConfig struct {
	// Add any configuration options here
	// For example, you might want to add:
	DefaultRepoPath string
	BlockedCommands []string
}

// NewGit creates and returns a new instance of the Git wrapper with the provided configuration.
func NewGit(logger observability.Logger, config GitConfig) *Git {
	return &Git{
		logger: logger,
		config: config,
	}
}

// GitAllInOneTool returns a mcp.Tool that can perform various Git operations
func (g *Git) GitAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        GitToolName,
		Description: "Performs any Git operation based on the provided command",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "Git command to execute"
				},
				"repo_path": {
					"type": "string",
					"description": "Path to Git repository"
				},
				"args": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "Arguments for the Git command"
				}
			},
			"required": ["command", "repo_path"]
		}`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
			span.SetAttributes(
				attribute.String("tool_name", params.Name),
				attribute.String("tool_argument", string(params.Arguments)),
			)
			defer span.End()

			g.logger.WithFields(map[string]interface{}{
				"tool_name": params.Name,
				"arguments": string(params.Arguments),
			}).Info("Starting git command execution")

			var input struct {
				Command  string   `json:"command"`
				RepoPath string   `json:"repo_path"`
				Args     []string `json:"args"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				g.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"raw_input":                 string(params.Arguments),
				}).Error("Failed to unmarshal input parameters")

				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
			}

			args := append([]string{"-C", input.RepoPath, input.Command}, input.Args...)
			cmd := exec.CommandContext(ctx, "git", args...)

			g.logger.WithFields(map[string]interface{}{
				"command":   input.Command,
				"repo_path": input.RepoPath,
				"args":      args,
			}).Debug("Executing git command")

			output, err := cmd.CombinedOutput()
			if err != nil {
				g.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"output":                    string(output),
					"command":                   input.Command,
				}).Error("Git command failed")

				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("git %s error: %w\nOutput: %s", input.Command, err, string(output))
			}

			g.logger.WithFields(map[string]interface{}{
				"command": input.Command,
				"output":  string(output),
			}).Debug("Git command completed successfully")

			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{
					Type: "text",
					Text: string(output),
				}},
			}, nil
		},
	}
}
