package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewGit(t *testing.T) {
	// Arrange
	logger := new(MockLogger)
	config := GitConfig{
		DefaultRepoPath: "/test/path",
		BlockedCommands: []string{"rm", "reset"},
	}

	// Act
	git := NewGit(logger, config)

	// Assert
	assert.NotNil(t, git)
	assert.Equal(t, logger, git.logger)
	assert.Equal(t, config, git.config)
}

func TestGit_GitAllInOneTool(t *testing.T) {
	// Arrange
	logger := new(MockLogger)
	git := NewGit(logger, GitConfig{})

	// Act
	tool := git.GitAllInOneTool()

	// Assert
	assert.Equal(t, GitToolName, tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.NotNil(t, tool.Handler)

	// Test input schema validation
	var schema map[string]interface{}
	err := json.Unmarshal(tool.InputSchema, &schema)
	assert.NoError(t, err)
	assert.Equal(t, "object", schema["type"])
}
func TestGit_GitAllInOneTool_Handler(t *testing.T) {
	tests := []struct {
		name          string
		input         mcp.CallToolParams
		expectedError bool
		setup         func(t *testing.T) (string, func())
	}{
		{
			name: "Valid git status command",
			input: mcp.CallToolParams{
				Name: GitToolName,
				Arguments: json.RawMessage(`{
					"command": "status",
					"repo_path": "test-repo",
					"args": []
				}`),
			},
			expectedError: false,
			setup: func(t *testing.T) (string, func()) {
				tmpDir, err := os.MkdirTemp("", "git-test-*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}

				repoPath := filepath.Join(tmpDir, "test-repo")
				err = os.MkdirAll(repoPath, 0755)
				if err != nil {
					t.Fatalf("Failed to create repo dir: %v", err)
				}

				runGitCommand := func(args ...string) {
					cmd := exec.Command("git", args...)
					cmd.Dir = repoPath
					output, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatalf("Git command failed: %v\nCommand: git %v\nOutput: %s",
							err, args, string(output))
					}
				}

				runGitCommand("init")
				runGitCommand("config", "user.email", "test@example.com")
				runGitCommand("config", "user.name", "Test User")
				runGitCommand("config", "commit.gpgsign", "false")

				testFile := filepath.Join(repoPath, "test.txt")
				if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}

				runGitCommand("add", ".")
				runGitCommand("commit", "-m", "Initial commit")

				return tmpDir, func() {
					if err := os.RemoveAll(tmpDir); err != nil {
						t.Errorf("Cleanup failed: %v", err)
					}
				}
			},
		},
		{
			name: "Invalid JSON input",
			input: mcp.CallToolParams{
				Name:      GitToolName,
				Arguments: json.RawMessage(`{invalid json}`),
			},
			expectedError: true,
			setup: func(t *testing.T) (string, func()) {
				return "", func() {}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := tt.setup(t)
			defer cleanup()

			if t.Failed() {
				return
			}

			logger := new(MockLogger)

			// Setup minimal logger expectations
			logger.On("WithFields", mock.Anything).Return(logger).Maybe()
			logger.On("Debug", mock.Anything).Return().Maybe()
			logger.On("Info", mock.Anything).Return().Maybe()
			logger.On("Error", mock.Anything).Return().Maybe()

			if !tt.expectedError {
				tt.input.Arguments = json.RawMessage(fmt.Sprintf(`{
					"command": "status",
					"repo_path": "%s",
					"args": []
				}`, filepath.Join(tmpDir, "test-repo")))
			}

			git := NewGit(logger, GitConfig{
				DefaultRepoPath: tmpDir,
			})
			tool := git.GitAllInOneTool()

			result, err := tool.Handler(context.Background(), tt.input)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.NotEmpty(t, result.Content)
			}
		})
	}
}
