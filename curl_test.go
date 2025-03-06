package mcptools

import (
	"context"
	"encoding/json"
	"github.com/shaharia-lab/goai/mcp"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCurl_CurlAllInOneTool(t *testing.T) {
	// Create a test server to handle our requests
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Echo back some response data that we can verify
		response := map[string]interface{}{
			"method":  r.Method,
			"headers": r.Header,
		}

		// Read body if present
		if r.Body != nil {
			defer r.Body.Close()
			var bodyData map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&bodyData); err == nil {
				response["body"] = bodyData
			}
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	tests := []struct {
		name     string
		input    map[string]interface{}
		wantErr  bool
		validate func(*testing.T, mcp.CallToolResult)
	}{
		{
			name: "basic GET request",
			input: map[string]interface{}{
				"url":    ts.URL,
				"method": "GET",
			},
			wantErr: false,
			validate: func(t *testing.T, result mcp.CallToolResult) {
				if len(result.Content) == 0 {
					t.Error("expected content in result")
				}
			},
		},
		{
			name: "POST request with data",
			input: map[string]interface{}{
				"url":    ts.URL,
				"method": "POST",
				"data":   `{"test": "data"}`,
				"headers": map[string]string{
					"Content-Type": "application/json",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result mcp.CallToolResult) {
				if len(result.Content) == 0 {
					t.Error("expected content in result")
				}
			},
		},
		{
			name: "request with environment variable in header",
			input: map[string]interface{}{
				"url":    ts.URL,
				"method": "GET",
				"headers": map[string]string{
					"X-Test-Header": "${TEST_ENV_VAR}",
				},
			},
			wantErr: false,
			validate: func(t *testing.T, result mcp.CallToolResult) {
				if len(result.Content) == 0 {
					t.Error("expected content in result")
				}
			},
		},
	}

	curl := &Curl{}
	tool := curl.CurlAllInOneTool()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set test environment variable if needed
			if tt.name == "request with environment variable in header" {
				os.Setenv("TEST_ENV_VAR", "test-value")
				defer os.Unsetenv("TEST_ENV_VAR")
			}

			// Convert input to JSON
			inputJSON, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to marshal input: %v", err)
			}

			// Call the tool
			result, err := tool.Handler(context.Background(), mcp.CallToolParams{
				Arguments: inputJSON,
			})

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("CurlAllInOneTool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Run custom validation
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}
