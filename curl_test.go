package mcptools

import (
	"context"
	"encoding/json"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCurl_Integration(t *testing.T) {
	// Create test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"method":  r.Method,
			"headers": r.Header,
			"path":    r.URL.Path,
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer ts.Close()

	tests := []struct {
		name           string
		config         CurlConfig
		input          map[string]interface{}
		envVars        map[string]string
		wantErr        bool
		errorContains  string
		checkLogs      func(t *testing.T, logger *observability.MockLogger)
		validateResult func(t *testing.T, result mcp.CallToolResult)
	}{
		{
			name: "successful GET request",
			config: CurlConfig{
				BlockedMethods: nil,
			},
			input: map[string]interface{}{
				"url":    ts.URL + "/test",
				"method": "GET",
				"headers": map[string]string{
					"X-Test-Header": "test-value",
				},
			},
			checkLogs: func(t *testing.T, logger *observability.MockLogger) {
				infoLogs := logger.GetInfoLogs()
				if len(infoLogs) == 0 {
					t.Error("Expected info logs but got none")
				}
			},
			validateResult: func(t *testing.T, result mcp.CallToolResult) {
				if len(result.Content) == 0 {
					t.Error("Expected non-empty result content")
				}
			},
		},
		{
			name: "blocked DELETE method",
			config: CurlConfig{
				BlockedMethods: []string{"DELETE"},
			},
			input: map[string]interface{}{
				"url":    ts.URL,
				"method": "DELETE",
			},
			wantErr:       true,
			errorContains: "is blocked",
			checkLogs: func(t *testing.T, logger *observability.MockLogger) {
				errorLogs := logger.GetErrorLogs()
				if len(errorLogs) == 0 {
					t.Error("Expected error logs for blocked method")
				}
			},
		},
		{
			name: "case insensitive blocked method",
			config: CurlConfig{
				BlockedMethods: []string{"DELETE"},
			},
			input: map[string]interface{}{
				"url":    ts.URL,
				"method": "delete",
			},
			wantErr:       true,
			errorContains: "is blocked",
		},
		{
			name: "invalid URL",
			config: CurlConfig{
				BlockedMethods: nil,
			},
			input: map[string]interface{}{
				"url":    "://invalid-url",
				"method": "GET",
			},
			wantErr:       true,
			errorContains: "invalid URL",
			checkLogs: func(t *testing.T, logger *observability.MockLogger) {
				errorLogs := logger.GetErrorLogs()
				if len(errorLogs) == 0 {
					t.Error("Expected error logs for invalid URL")
				}
			},
		},
		{
			name: "POST with data",
			config: CurlConfig{
				BlockedMethods: nil,
			},
			input: map[string]interface{}{
				"url":    ts.URL,
				"method": "POST",
				"data":   `{"test":"value"}`,
				"headers": map[string]string{
					"Content-Type": "application/json",
				},
			},
			validateResult: func(t *testing.T, result mcp.CallToolResult) {
				if len(result.Content) == 0 {
					t.Error("Expected non-empty result content")
				}
			},
		},
		{
			name: "with environment variables in headers",
			config: CurlConfig{
				BlockedMethods: nil,
			},
			input: map[string]interface{}{
				"url":    ts.URL,
				"method": "GET",
				"headers": map[string]string{
					"X-Auth-Token": "${TEST_AUTH_TOKEN}",
				},
			},
			envVars: map[string]string{
				"TEST_AUTH_TOKEN": "secret-token-value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			curl := NewCurl(observability.NewMockLogger(), tt.config)
			tool := curl.CurlAllInOneTool()

			inputJSON, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			result, err := tool.Handler(context.Background(), mcp.CallToolParams{
				Name:      "curl_all_in_one",
				Arguments: inputJSON,
			})

			// Check error expectations
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Error %q does not contain expected message %q", err.Error(), tt.errorContains)
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Run custom result validation
			if tt.validateResult != nil && !tt.wantErr {
				tt.validateResult(t, result)
			}
		})
	}
}

func TestCurl_Configuration(t *testing.T) {
	tests := []struct {
		name           string
		blockedMethods []string
		testMethod     string
		shouldBlock    bool
	}{
		{
			name:           "empty blocked methods",
			blockedMethods: []string{},
			testMethod:     "DELETE",
			shouldBlock:    false,
		},
		{
			name:           "single blocked method",
			blockedMethods: []string{"DELETE"},
			testMethod:     "DELETE",
			shouldBlock:    true,
		},
		{
			name:           "case insensitive blocking",
			blockedMethods: []string{"DELETE"},
			testMethod:     "delete",
			shouldBlock:    true,
		},
		{
			name:           "multiple blocked methods",
			blockedMethods: []string{"DELETE", "PURGE", "TRACE"},
			testMethod:     "PURGE",
			shouldBlock:    true,
		},
		{
			name:           "non-blocked method",
			blockedMethods: []string{"DELETE", "PURGE"},
			testMethod:     "GET",
			shouldBlock:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			curl := NewCurl(observability.NewMockLogger(), CurlConfig{
				BlockedMethods: tt.blockedMethods,
			})

			isBlocked := curl.isMethodBlocked(tt.testMethod)
			if isBlocked != tt.shouldBlock {
				t.Errorf("isMethodBlocked(%q) = %v, want %v", tt.testMethod, isBlocked, tt.shouldBlock)
			}
		})
	}
}
