package mcptools

import (
	"context"
	"encoding/json"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"os/exec"
	"testing"
)

func TestNewCurl(t *testing.T) {
	tests := []struct {
		name            string
		config          CurlConfig
		expectedBlocked []string
	}{
		{
			name: "Empty blocked methods",
			config: CurlConfig{
				BlockedMethods: []string{},
			},
			expectedBlocked: []string{},
		},
		{
			name: "Multiple blocked methods",
			config: CurlConfig{
				BlockedMethods: []string{"POST", "delete", "PUT"},
			},
			expectedBlocked: []string{"POST", "DELETE", "PUT"},
		},
		{
			name: "Case normalization",
			config: CurlConfig{
				BlockedMethods: []string{"post", "Delete", "PUT"},
			},
			expectedBlocked: []string{"POST", "DELETE", "PUT"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)
			mockLogger.On("WithFields", mock.Anything).Return(mockLogger)

			curl := NewCurl(mockLogger, tt.config)

			assert.NotNil(t, curl)
			assert.Equal(t, len(tt.expectedBlocked), len(curl.blockedMethods))
			for i, method := range curl.blockedMethods {
				assert.Equal(t, tt.expectedBlocked[i], method)
			}
		})
	}
}

func TestCurl_isMethodBlocked(t *testing.T) {
	tests := []struct {
		name           string
		blockedMethods []string
		testMethod     string
		expected       bool
	}{
		{
			name:           "Blocked method - exact match",
			blockedMethods: []string{"POST", "DELETE"},
			testMethod:     "POST",
			expected:       true,
		},
		{
			name:           "Blocked method - case insensitive",
			blockedMethods: []string{"POST", "DELETE"},
			testMethod:     "post",
			expected:       true,
		},
		{
			name:           "Not blocked method",
			blockedMethods: []string{"POST", "DELETE"},
			testMethod:     "GET",
			expected:       false,
		},
		{
			name:           "Empty blocked list",
			blockedMethods: []string{},
			testMethod:     "POST",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLogger := new(MockLogger)
			curl := NewCurl(mockLogger, CurlConfig{BlockedMethods: tt.blockedMethods})

			result := curl.isMethodBlocked(tt.testMethod)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// MockCommandExecutor implements CommandExecutor for testing
type MockCommandExecutor struct {
	mock.Mock
}

func (m *MockCommandExecutor) ExecuteCommand(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	args := m.Called(ctx, cmd)
	return args.Get(0).([]byte), args.Error(1)
}

// curl_test.go

func TestCurl_CurlAllInOneTool(t *testing.T) {
	mockLogger := new(MockLogger)
	mockExecutor := new(MockCommandExecutor)

	// Set up mock expectations for logger
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything).Return()

	// Set up mock expectations for executor
	mockExecutor.On("ExecuteCommand", mock.Anything, mock.Anything).Return(
		[]byte("mock response"), nil,
	)

	curl := NewCurl(mockLogger, CurlConfig{BlockedMethods: []string{"DELETE"}})
	curl.cmdExecutor = mockExecutor

	tool := curl.CurlAllInOneTool()

	// Test tool metadata
	assert.Equal(t, CurlToolName, tool.Name)
	assert.NotEmpty(t, tool.Description)

	// Validate input schema
	var schema map[string]interface{}
	err := json.Unmarshal(tool.InputSchema, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "object", schema["type"])

	// Test handler with valid input
	validInput := map[string]interface{}{
		"url":    "https://api.example.com",
		"method": "GET",
	}
	inputJSON, _ := json.Marshal(validInput)

	ctx := context.Background()
	result, err := tool.Handler(ctx, mcp.CallToolParams{
		Name:      CurlToolName,
		Arguments: inputJSON,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []mcp.ToolResultContent{
		{Type: "text", Text: "mock response"},
	}, result.Content)

	mockLogger.AssertExpectations(t)
	mockExecutor.AssertExpectations(t)
}

func TestCurl_HandlerValidation(t *testing.T) {
	mockLogger := new(MockLogger)
	mockExecutor := new(MockCommandExecutor)

	// Set up mock logger expectations
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything).Return()

	curl := NewCurl(mockLogger, CurlConfig{BlockedMethods: []string{"DELETE"}})
	curl.cmdExecutor = mockExecutor

	tool := curl.CurlAllInOneTool()

	tests := []struct {
		name          string
		input         map[string]interface{}
		expectedError string
	}{
		{
			name: "Invalid URL",
			input: map[string]interface{}{
				"url":    "http://[invalid-url", // Use a definitely invalid URL format
				"method": "GET",
			},
			expectedError: "invalid URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inputJSON, _ := json.Marshal(tt.input)

			// Don't set up any expectations for ExecuteCommand
			// If it gets called, the test will fail automatically

			result, err := tool.Handler(context.Background(), mcp.CallToolParams{
				Name:      "curl",
				Arguments: inputJSON,
			})

			assert.Empty(t, result.Content)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
			mockExecutor.AssertNotCalled(t, "ExecuteCommand")
		})
	}

	mockLogger.AssertExpectations(t)
}
