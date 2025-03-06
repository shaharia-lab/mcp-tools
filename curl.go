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

// Curl represents a wrapper around the system's curl command-line tool,
// providing a programmatic interface for making HTTP requests.
type Curl struct {
	logger         observability.Logger
	blockedMethods []string // List of HTTP methods that are not allowed
}

// CurlConfig holds the configuration for the Curl tool
type CurlConfig struct {
	BlockedMethods []string
}

// NewCurl creates and returns a new instance of the Curl wrapper with the provided configuration.
func NewCurl(logger observability.Logger, config CurlConfig) *Curl {
	// Convert blocked methods to uppercase for case-insensitive comparison
	blockedMethods := make([]string, len(config.BlockedMethods))
	for i, method := range config.BlockedMethods {
		blockedMethods[i] = strings.ToUpper(method)
	}

	return &Curl{
		logger:         logger,
		blockedMethods: blockedMethods,
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
		Name:        "curl_all_in_one",
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
			c.logger.Info("Starting curl request execution",
				"tool_name", params.Name,
				"arguments", string(params.Arguments),
				"timestamp", startTime.Format(time.RFC3339),
			)

			var input struct {
				URL      string            `json:"url"`
				Method   string            `json:"method"`
				Data     string            `json:"data"`
				Headers  map[string]string `json:"headers"`
				Insecure bool              `json:"insecure"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				c.logger.Error("Failed to unmarshal input parameters",
					"error", err,
					"raw_input", string(params.Arguments),
				)
				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("failed to parse input: %w", err)
			}

			// Validate URL
			parsedURL, err := url.Parse(input.URL)
			if err != nil {
				c.logger.Error("Invalid URL provided",
					"url", input.URL,
					"error", err,
				)
				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("invalid URL: %w", err)
			}

			// Check if method is blocked
			if c.isMethodBlocked(input.Method) {
				err := fmt.Errorf("HTTP method %s is blocked", input.Method)
				c.logger.Error("Blocked HTTP method attempted",
					"method", input.Method,
					"url", input.URL,
				)
				span.RecordError(err)
				return mcp.CallToolResult{}, err
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

			// Log the full command (excluding sensitive data)
			c.logger.Debug("Executing curl command",
				"method", input.Method,
				"url", input.URL,
				"headers_count", len(input.Headers),
				"has_data", input.Data != "",
				"insecure", input.Insecure,
			)

			// Execute the command
			cmd := exec.CommandContext(ctx, "curl", args...)
			output, err := cmd.CombinedOutput()

			// Log execution results
			executionTime := time.Since(startTime)
			if err != nil {
				c.logger.Error("Curl command failed",
					"error", err,
					"output", string(output),
					"duration_ms", executionTime.Milliseconds(),
				)
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
