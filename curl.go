// Package mcptools provides tools for making HTTP requests using curl
package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/goai/mcp"
	"log"
	"os"
	"os/exec"
	"strings"
)

// Curl represents a wrapper around the system's curl command-line tool,
// providing a programmatic interface for making HTTP requests.
type Curl struct {
}

// NewCurl creates and returns a new instance of the Curl wrapper.
// This is a constructor method that initializes a new Curl object.
func (c *Curl) NewCurl() *Curl {
	return &Curl{}
}

// CurlAllInOneTool returns a mcp.Tool that can perform various HTTP requests
// using the system's curl command. It supports all standard HTTP methods,
// custom headers, request body data, and SSL configuration.
//
// The tool accepts the following JSON input parameters:
//   - url: (required) The target URL for the HTTP request
//   - method: (required) The HTTP method to use (GET, POST, PUT, DELETE, PATCH, etc.)
//   - data: (optional) The request body data to send
//   - headers: (optional) A map of HTTP headers to include in the request
//   - insecure: (optional) Boolean flag to allow insecure SSL connections
//
// The tool supports environment variable expansion in header values using ${VAR} syntax.
// It returns the command output as a text result in the mcp.CallToolResult structure.
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
			var input struct {
				URL      string            `json:"url"`
				Method   string            `json:"method"`
				Data     string            `json:"data"`
				Headers  map[string]string `json:"headers"`
				Insecure bool              `json:"insecure"`
			}
			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return mcp.CallToolResult{}, err
			}

			// Replace environment variable placeholders in headers
			for key, value := range input.Headers {
				input.Headers[key] = os.ExpandEnv(value)
			}

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

			marshal, err := json.Marshal(input)
			if err != nil {
				return mcp.CallToolResult{}, err
			}

			log.Printf("curl %s", marshal)

			cmd := exec.CommandContext(ctx, "curl", args...)
			output, err := cmd.CombinedOutput()
			if err != nil {
				return mcp.CallToolResult{}, err
			}

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
