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

	// Mock WithFields expectation
	mockLogger.On("WithFields", mock.MatchedBy(func(fields map[string]interface{}) bool {
		tool, ok := fields["tool"]
		return ok && tool == "docker"
	})).Return(mockLogger)

	// Mock Info expectation for "Received input"
	mockLogger.On("Info",
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 3 &&
				args[0] == "Received input" &&
				args[1] == "input"
		}),
	).Return()

	// Mock Info expectation for "Executing docker command"
	mockLogger.On("Info",
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 5 &&
				args[0] == "Executing docker command" &&
				args[1] == "command" &&
				args[2] == "ps" &&
				args[3] == "args"
		}),
	).Return()

	// Mock Info expectation for "Docker command executed successfully"
	mockLogger.On("Info",
		mock.MatchedBy(func(args []interface{}) bool {
			return len(args) == 5 &&
				args[0] == "Docker command executed successfully" &&
				args[1] == "command" &&
				args[2] == "ps" &&
				args[3] == "args"
		}),
	).Return()

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
