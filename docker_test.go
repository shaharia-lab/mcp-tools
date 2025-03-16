package mcptools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewDocker(t *testing.T) {
	mockLogger := new(MockLogger)
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)

	docker := NewDocker(mockLogger)

	assert.NotNil(t, docker)
	assert.NotNil(t, docker.cmdExecutor)
	assert.NotNil(t, docker.logger)
}

func TestDocker_DockerAllInOneTool(t *testing.T) {
	mockLogger := new(MockLogger)
	mockExecutor := new(MockCommandExecutor)

	mockExecutor.On("ExecuteCommand", mock.Anything, mock.Anything).Return(
		[]byte("mock docker output"), nil,
	)

	docker := NewDocker(mockLogger)
	docker.cmdExecutor = mockExecutor

	tool := docker.DockerAllInOneTool()

	// Test tool metadata
	assert.Equal(t, DockerToolName, tool.Name)
	assert.NotEmpty(t, tool.Description)

	// Validate input schema
	var schema map[string]interface{}
	err := json.Unmarshal(tool.InputSchema, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "object", schema["type"])

	// Test handler with valid input
	validInput := map[string]interface{}{
		"command": "ps",
		"args":    []string{"-a"},
	}
	inputJSON, _ := json.Marshal(validInput)

	result, err := tool.Handler(context.Background(), mcp.CallToolParams{
		Name:      DockerToolName,
		Arguments: inputJSON,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, []mcp.ToolResultContent{
		{Type: "text", Text: "mock docker output"},
	}, result.Content)

	mockExecutor.AssertExpectations(t)
}
func TestDocker_HandlerValidation(t *testing.T) {
	mockLogger := new(MockLogger)
	mockExecutor := new(MockCommandExecutor)

	// Set up mock logger expectations
	mockLogger.On("WithFields", mock.MatchedBy(func(fields map[string]interface{}) bool {
		_, hasError := fields["error"]
		return hasError
	})).Return(mockLogger)

	// Change this line to match the actual method signature
	mockLogger.On("Error", mock.Anything).Return()

	docker := NewDocker(mockLogger)
	docker.cmdExecutor = mockExecutor

	tool := docker.DockerAllInOneTool()

	tests := []struct {
		name          string
		input         map[string]interface{}
		expectedError string
	}{
		{
			name: "Missing command",
			input: map[string]interface{}{
				"args": []string{"-a"},
			},
			expectedError: "command is required",
		},
		{
			name: "Empty command",
			input: map[string]interface{}{
				"command": "",
				"args":    []string{"-a"},
			},
			expectedError: "command is required",
		},
		{
			name: "Invalid JSON",
			input: map[string]interface{}{
				"command": "test",
			},
			expectedError: "failed to parse input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputJSON []byte
			if tt.name == "Invalid JSON" {
				inputJSON = []byte(`{invalid json}`) // Deliberately malformed JSON
			} else {
				inputJSON, _ = json.Marshal(tt.input)
			}

			result, err := tool.Handler(context.Background(), mcp.CallToolParams{
				Name:      DockerToolName,
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

func TestDocker_ValidateDockerInput(t *testing.T) {
	tests := []struct {
		name  string
		input struct {
			Command string   `json:"command"`
			Args    []string `json:"args"`
		}
		expectError bool
	}{
		{
			name: "Valid input",
			input: struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			}{
				Command: "ps",
				Args:    []string{"-a"},
			},
			expectError: false,
		},
		{
			name: "Empty command",
			input: struct {
				Command string   `json:"command"`
				Args    []string `json:"args"`
			}{
				Command: "",
				Args:    []string{"-a"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDockerInput(tt.input)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
