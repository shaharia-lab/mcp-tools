package mcptools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"

	"github.com/shaharia-lab/goai"
)

const GrepToolName = "grep"

// Grep represents a wrapper around the system's grep command-line tool
type Grep struct {
	logger      goai.Logger
	cmdExecutor CommandExecutor
}

// NewGrep creates and returns a new instance of the Grep wrapper
func NewGrep(logger goai.Logger) *Grep {
	return &Grep{
		logger:      logger,
		cmdExecutor: &RealCommandExecutor{},
	}
}

// GrepAllInOneTool returns a goai.Tool that can execute grep commands
func (g *Grep) GrepAllInOneTool() goai.Tool {
	return goai.Tool{
		Name:        GrepToolName,
		Description: "Execute grep commands with specified pattern and options",
		InputSchema: json.RawMessage(`{
            "type": "object",
            "properties": {
                "pattern": {
                    "type": "string",
                    "description": "Pattern to search for"
                },
                "path": {
                    "type": "string",
                    "description": "File or directory path to search in"
                },
                "options": {
                    "type": "array",
                    "items": {
                        "type": "string"
                    },
                    "description": "Additional grep options (e.g., -r for recursive, -i for case-insensitive)"
                }
            },
            "required": ["pattern", "path"]
        }`),
		Handler: func(ctx context.Context, params goai.CallToolParams) (goai.CallToolResult, error) {
			ctx, span := goai.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
			defer span.End()

			var input struct {
				Pattern string   `json:"pattern"`
				Path    string   `json:"path"`
				Options []string `json:"options"`
			}

			g.logger.WithFields(map[string]interface{}{
				"tool_name": params.Name,
				"arguments": string(params.Arguments),
			}).Info("Executing grep command")

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				g.logger.WithFields(map[string]interface{}{
					goai.ErrorLogField: err,
					"raw_input":                 string(params.Arguments),
				}).Error("Failed to unmarshal input parameters")
				span.RecordError(err)
				return goai.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			if err := validateGrepInput(input); err != nil {
				g.logger.WithFields(map[string]interface{}{
					goai.ErrorLogField: err,
				}).Error("Input validation failed")
				span.RecordError(err)
				return returnErrorOutput(err), nil
			}

			// Ensure recursive search is enabled if a directory is provided
			hasRecursive := false
			for _, opt := range input.Options {
				if opt == "-r" || opt == "-R" {
					hasRecursive = true
					break
				}
			}
			if !hasRecursive {
				input.Options = append(input.Options, "-r")
			}

			args := append(input.Options, "-E")
			args = append(args, input.Pattern, input.Path)

			g.logger.WithFields(map[string]interface{}{
				"tool":    GrepToolName,
				"command": "grep",
				"args":    args,
			}).Info("Executing grep command", "args", args)

			cmd := exec.Command("grep", args...)

			// Execute the command using the executor
			output, err := g.cmdExecutor.ExecuteCommand(ctx, cmd)

			// Special handling for grep exit codes
			if err != nil {
				var exitError *exec.ExitError
				if errors.As(err, &exitError) {
					// Exit code 1 means no matches found (not an error)
					if exitError.ExitCode() == 1 {
						return goai.CallToolResult{
							Content: []goai.ToolResultContent{
								{
									Type: "text",
									Text: "No matches found",
								},
							},
							IsError: false,
						}, nil
					}
					// Exit code 2 or others indicate real errors
					errorMsg := string(exitError.Stderr)
					if errorMsg == "" {
						errorMsg = err.Error()
					}

					g.logger.WithFields(map[string]interface{}{
						goai.ErrorLogField: err,
						"command":                   "grep",
						"args":                      args,
						"exit_code":                 exitError.ExitCode(),
						"stderr":                    errorMsg,
					}).Error("Grep command execution failed")

					return goai.CallToolResult{
						Content: []goai.ToolResultContent{
							{
								Type: "text",
								Text: fmt.Sprintf("Grep command failed (exit code %d): %s\nCommand: grep %v",
									exitError.ExitCode(), errorMsg, args),
							},
						},
						IsError: true,
					}, nil
				}
				// Handle non-exit errors
				return goai.CallToolResult{
					Content: []goai.ToolResultContent{
						{
							Type: "text",
							Text: fmt.Sprintf("Command execution error: %s", err.Error()),
						},
					},
					IsError: true,
				}, nil
			}

			g.logger.WithFields(map[string]interface{}{
				"tool":          GrepToolName,
				"output_lenght": len(string(output)),
			}).Info("Grep command executed successfully")

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

func validateGrepInput(input struct {
	Pattern string   `json:"pattern"`
	Path    string   `json:"path"`
	Options []string `json:"options"`
}) error {
	if input.Pattern == "" {
		return fmt.Errorf("pattern is required")
	}
	if input.Path == "" {
		return fmt.Errorf("path is required")
	}
	return nil
}
