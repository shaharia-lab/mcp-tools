package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"go.opentelemetry.io/otel/attribute"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"
)

const CurlToolName = "curl_all_in_one"

// Curl represents a wrapper around the system's curl command-line tool,
// providing a programmatic interface for making HTTP requests.
type Curl struct {
	logger         observability.Logger
	blockedMethods []string
	cmdExecutor    CommandExecutor
}

// CurlConfig holds the configuration for the Curl tool
type CurlConfig struct {
	BlockedMethods []string
}

// NewCurl creates and returns a new instance of the Curl wrapper with the provided configuration.
func NewCurl(logger observability.Logger, config CurlConfig) *Curl {
	blockedMethods := make([]string, len(config.BlockedMethods))
	for i, method := range config.BlockedMethods {
		blockedMethods[i] = strings.ToUpper(method)
	}

	return &Curl{
		logger:         logger,
		blockedMethods: blockedMethods,
		cmdExecutor:    &RealCommandExecutor{},
	}
}

// isMethodBlocked checks if the given HTTP method is in the blocked list
func (c *Curl) isMethodBlocked(method string) bool {
	method = strings.ToUpper(method)
	for _, blocked := range c.blockedMethods {
		if blocked == method {
			return true
		}
	}
	return false
}

// CurlAllInOneTool returns a mcp.Tool that can perform various HTTP requests
func (c *Curl) CurlAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        CurlToolName,
		Description: "Perform any HTTP request with specified method, URL, headers, and data",
		InputSchema: json.RawMessage(`{
        "type": "object",
        "properties": {
            "url": {
                "type": "string",
                "description": "Target URL for the request"
            },
            "method": {
                "type": "string",
                "description": "HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)"
            },
            "data": {
                "type": "string",
                "description": "Data to send in the request body"
            },
            "headers": {
                "type": "object",
                "description": "HTTP headers to include in the request",
                "additionalProperties": {
                    "type": "string"
                }
            },
            "insecure": {
                "type": "boolean",
                "description": "Allow insecure server connections when using SSL"
            }
        },
        "required": ["url", "method"]
    }`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			// Start tracing span
			ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
			defer span.End()

			startTime := time.Now()
			c.logger.WithFields(map[string]interface{}{
				"tool_name": params.Name,
				"arguments": string(params.Arguments),
				"timestamp": startTime.Format(time.RFC3339),
			}).Info("Starting curl request execution")

			var input struct {
				URL      string            `json:"url"`
				Method   string            `json:"method"`
				Data     string            `json:"data"`
				Headers  map[string]string `json:"headers"`
				Insecure bool              `json:"insecure"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				c.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"raw_input":                 string(params.Arguments),
				}).Error("Failed to unmarshal input parameters")

				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			// In your Handler function, add validation before command execution:
			if err := validateInput(input); err != nil {
				c.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
				}).Error("Input validation failed")
				span.RecordError(err)
				return mcp.CallToolResult{}, err
			}

			// Check blocked methods after basic validation
			if c.isMethodBlocked(input.Method) {
				err := fmt.Errorf("HTTP method %s is blocked", input.Method)
				c.logger.WithFields(map[string]interface{}{
					"method": input.Method,
					"url":    input.URL,
				}).Error("Blocked HTTP method attempted")
				span.RecordError(err)
				return mcp.CallToolResult{}, err
			}

			// Validate URL
			parsedURL, err := url.Parse(input.URL)
			if err != nil {
				c.logger.WithFields(map[string]interface{}{
					"url":                       input.URL,
					observability.ErrorLogField: err,
				}).Error("Invalid URL provided")

				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("invalid URL: %w", err)
			}

			// Set span attributes
			span.SetAttributes(
				attribute.String("http.method", input.Method),
				attribute.String("http.url", input.URL),
				attribute.String("http.scheme", parsedURL.Scheme),
				attribute.String("http.host", parsedURL.Host),
			)

			// Replace environment variable placeholders in headers
			for key, value := range input.Headers {
				input.Headers[key] = os.ExpandEnv(value)
			}

			// Build curl command arguments
			args := []string{"-s", "-X", strings.ToUpper(input.Method)}
			if input.Insecure {
				args = append(args, "-k")
			}

			for key, value := range input.Headers {
				args = append(args, "-H", fmt.Sprintf("%s: %s", key, value))
			}

			if input.Data != "" {
				args = append(args, "-d", input.Data)
			}

			args = append(args, input.URL)

			c.logger.WithFields(map[string]interface{}{
				"method":        input.Method,
				"url":           input.URL,
				"headers_count": len(input.Headers),
				"has_data":      input.Data != "",
				"insecure":      input.Insecure,
			}).Info("Executing curl command")

			// Execute the command
			cmd := exec.CommandContext(ctx, "curl", args...)
			output, err := c.cmdExecutor.ExecuteCommand(ctx, cmd)

			// Log execution results
			executionTime := time.Since(startTime)
			if err != nil {
				c.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"output":                    string(output),
					"duration_ms":               executionTime.Milliseconds(),
				}).Error("Curl command failed")
				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("curl command failed: %w", err)
			}

			c.logger.Info("Curl command completed successfully",
				"duration_ms", executionTime.Milliseconds(),
				"output_size", len(output),
			)

			// Set success span attributes
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

func validateInput(input struct {
	URL      string            `json:"url"`
	Method   string            `json:"method"`
	Data     string            `json:"data"`
	Headers  map[string]string `json:"headers"`
	Insecure bool              `json:"insecure"`
}) error {
	// Check required fields first
	if input.Method == "" {
		return fmt.Errorf("method is required")
	}

	if input.URL == "" {
		return fmt.Errorf("url is required")
	}

	// Validate URL format
	_, err := url.Parse(input.URL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	return nil
}
