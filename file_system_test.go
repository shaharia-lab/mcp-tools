package mcptools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fsTestSuite struct {
	t        *testing.T
	fs       *FileSystem
	tempDir  string
	testFile string
}

func setupFSTest(t *testing.T) *fsTestSuite {
	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	config := FileSystemConfig{
		AllowedDirectory: tempDir,
		BlockedPatterns:  []string{"*.exe", "*.dll"},
	}

	fs := NewFileSystem(&MockLogger{}, config)

	return &fsTestSuite{
		t:        t,
		fs:       fs,
		tempDir:  tempDir,
		testFile: testFile,
	}
}

func (s *fsTestSuite) cleanup() {
	os.RemoveAll(s.tempDir)
}

func TestFileSystem_List(t *testing.T) {
	mockLogger := &MockLogger{}
	// Set up base expectations for the logger
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything).Return()
	mockLogger.On("Error", mock.Anything).Return()

	// Allow WithErr to be called multiple times (zero or more)
	mockLogger.On("WithErr", mock.AnythingOfType("error")).Return(mockLogger).Maybe()

	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	fs := NewFileSystem(mockLogger, FileSystemConfig{
		AllowedDirectory: tempDir,
		BlockedPatterns:  []string{"*.exe", "*.dll"},
	})

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantErr   bool
		checkFunc func(t *testing.T, result mcp.CallToolResult)
	}{
		{
			name: "list directory success",
			input: map[string]interface{}{
				"operation": "list",
				"path":      tempDir,
				"recursive": false,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result mcp.CallToolResult) {
				assert.Contains(t, result.Content[0].Text, "test.txt")
			},
		},
		{
			name: "list non-existent directory",
			input: map[string]interface{}{
				"operation": "list",
				"path":      filepath.Join(tempDir, "nonexistent"),
			},
			wantErr: true,
		},
		{
			name: "list outside allowed directory",
			input: map[string]interface{}{
				"operation": "list",
				"path":      "/",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result, err := fs.FileSystemAllInOneTool().Handler(context.Background(), mcp.CallToolParams{
				Name:      FileSystemToolName,
				Arguments: args,
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}

	// Verify that all expected mock calls were made
	mockLogger.AssertExpectations(t)
}

func TestFileSystem_Read(t *testing.T) {
	mockLogger := &MockLogger{}
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything).Return()
	mockLogger.On("Error", mock.Anything).Return()
	mockLogger.On("WithErr", mock.AnythingOfType("error")).Return(mockLogger).Maybe()

	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	fs := NewFileSystem(mockLogger, FileSystemConfig{
		AllowedDirectory: tempDir,
		BlockedPatterns:  []string{"*.exe", "*.dll"},
	})

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantErr   bool
		checkFunc func(t *testing.T, result mcp.CallToolResult)
	}{
		{
			name: "read file success",
			input: map[string]interface{}{
				"operation": "read",
				"path":      testFile,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result mcp.CallToolResult) {
				assert.Equal(t, "test content", result.Content[0].Text)
			},
		},
		{
			name: "read non-existent file",
			input: map[string]interface{}{
				"operation": "read",
				"path":      filepath.Join(tempDir, "nonexistent.txt"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result, err := fs.FileSystemAllInOneTool().Handler(context.Background(), mcp.CallToolParams{
				Name:      FileSystemToolName,
				Arguments: args,
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}

	mockLogger.AssertExpectations(t)
}

func TestFileSystem_Write(t *testing.T) {
	// Create mock logger with proper expectations
	mockLogger := &MockLogger{}
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything).Return()
	mockLogger.On("Error", mock.Anything).Return()
	mockLogger.On("WithErr", mock.AnythingOfType("error")).Return(mockLogger).Maybe()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := NewFileSystem(mockLogger, FileSystemConfig{
		AllowedDirectory: tempDir,
		BlockedPatterns:  []string{"*.exe", "*.dll"},
	})

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantErr   bool
		checkFunc func(t *testing.T)
	}{
		{
			name: "write file success",
			input: map[string]interface{}{
				"operation": "write",
				"path":      filepath.Join(tempDir, "new.txt"),
				"content":   "new content",
			},
			wantErr: false,
			checkFunc: func(t *testing.T) {
				content, err := os.ReadFile(filepath.Join(tempDir, "new.txt"))
				require.NoError(t, err)
				assert.Equal(t, "new content", string(content))
			},
		},
		{
			name: "write blocked extension",
			input: map[string]interface{}{
				"operation": "write",
				"path":      filepath.Join(tempDir, "test.exe"),
				"content":   "blocked content",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := json.Marshal(tt.input)
			require.NoError(t, err)

			_, err = fs.FileSystemAllInOneTool().Handler(context.Background(), mcp.CallToolParams{
				Name:      FileSystemToolName,
				Arguments: args,
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t)
			}
		})
	}

	mockLogger.AssertExpectations(t)
}

func TestFileSystem_Delete(t *testing.T) {
	// Create mock logger with proper expectations
	mockLogger := &MockLogger{}
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything).Return()
	mockLogger.On("Error", mock.Anything).Return().Maybe()
	mockLogger.On("WithErr", mock.AnythingOfType("error")).Return(mockLogger).Maybe()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := NewFileSystem(mockLogger, FileSystemConfig{
		AllowedDirectory: tempDir,
		BlockedPatterns:  []string{"*.exe", "*.dll"},
	})

	tests := []struct {
		name      string
		input     map[string]interface{}
		setup     func(t *testing.T) string
		wantErr   bool
		checkFunc func(t *testing.T, path string)
	}{
		{
			name: "delete file success",
			setup: func(t *testing.T) string {
				path := filepath.Join(tempDir, "to_delete.txt")
				require.NoError(t, os.WriteFile(path, []byte("delete me"), 0644))
				return path
			},
			input: map[string]interface{}{
				"operation": "delete",
				"path":      "${path}",
			},
			wantErr: false,
			checkFunc: func(t *testing.T, path string) {
				_, err := os.Stat(path)
				assert.True(t, os.IsNotExist(err))
			},
		},
		{
			name: "delete directory recursively",
			setup: func(t *testing.T) string {
				path := filepath.Join(tempDir, "subdir")
				require.NoError(t, os.MkdirAll(path, 0755))
				require.NoError(t, os.WriteFile(filepath.Join(path, "file.txt"), []byte("content"), 0644))
				return path
			},
			input: map[string]interface{}{
				"operation": "delete",
				"path":      "${path}",
				"recursive": true,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, path string) {
				_, err := os.Stat(path)
				assert.True(t, os.IsNotExist(err))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			if tt.setup != nil {
				path = tt.setup(t)
				// Replace ${path} placeholder with actual path
				tt.input["path"] = path
			}

			args, err := json.Marshal(tt.input)
			require.NoError(t, err)

			_, err = fs.FileSystemAllInOneTool().Handler(context.Background(), mcp.CallToolParams{
				Name:      FileSystemToolName,
				Arguments: args,
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, path)
			}
		})
	}

	mockLogger.AssertExpectations(t)
}

func TestFileSystem_Tree(t *testing.T) {
	// Create mock logger with proper expectations
	mockLogger := &MockLogger{}
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything).Return()
	mockLogger.On("Error", mock.Anything).Return().Maybe()
	mockLogger.On("WithErr", mock.AnythingOfType("error")).Return(mockLogger).Maybe()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a directory structure for testing
	require.NoError(t, os.MkdirAll(filepath.Join(tempDir, "dir1/subdir"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "dir1/file1.txt"), []byte("content"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tempDir, "dir1/subdir/file2.txt"), []byte("content"), 0644))

	fs := NewFileSystem(mockLogger, FileSystemConfig{
		AllowedDirectory: tempDir,
		BlockedPatterns:  []string{"*.exe", "*.dll"},
	})

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantErr   bool
		checkFunc func(t *testing.T, result mcp.CallToolResult)
	}{
		{
			name: "tree directory success",
			input: map[string]interface{}{
				"operation": "tree",
				"path":      tempDir,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result mcp.CallToolResult) {
				assert.Contains(t, result.Content[0].Text, "dir1")
				assert.Contains(t, result.Content[0].Text, "subdir")
				assert.Contains(t, result.Content[0].Text, "file1.txt")
				assert.Contains(t, result.Content[0].Text, "file2.txt")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result, err := fs.FileSystemAllInOneTool().Handler(context.Background(), mcp.CallToolParams{
				Name:      FileSystemToolName,
				Arguments: args,
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}

	mockLogger.AssertExpectations(t)
}

func TestFileSystem_Mkdir(t *testing.T) {
	// Create mock logger with proper expectations
	mockLogger := &MockLogger{}
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything).Return()
	mockLogger.On("Error", mock.Anything).Return().Maybe()
	mockLogger.On("WithErr", mock.AnythingOfType("error")).Return(mockLogger).Maybe()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	fs := NewFileSystem(mockLogger, FileSystemConfig{
		AllowedDirectory: tempDir,
		BlockedPatterns:  []string{"*.exe", "*.dll"},
	})

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantErr   bool
		checkFunc func(t *testing.T)
	}{
		{
			name: "create directory success",
			input: map[string]interface{}{
				"operation": "mkdir",
				"path":      filepath.Join(tempDir, "newdir"),
			},
			wantErr: false,
			checkFunc: func(t *testing.T) {
				info, err := os.Stat(filepath.Join(tempDir, "newdir"))
				require.NoError(t, err)
				assert.True(t, info.IsDir())
			},
		},
		{
			name: "create nested directory",
			input: map[string]interface{}{
				"operation": "mkdir",
				"path":      filepath.Join(tempDir, "nested/dir"),
			},
			wantErr: false,
			checkFunc: func(t *testing.T) {
				info, err := os.Stat(filepath.Join(tempDir, "nested/dir"))
				require.NoError(t, err)
				assert.True(t, info.IsDir())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := json.Marshal(tt.input)
			require.NoError(t, err)

			_, err = fs.FileSystemAllInOneTool().Handler(context.Background(), mcp.CallToolParams{
				Name:      FileSystemToolName,
				Arguments: args,
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t)
			}
		})
	}

	mockLogger.AssertExpectations(t)
}

func TestFileSystem_Search(t *testing.T) {
	// Create mock logger with proper expectations
	mockLogger := &MockLogger{}
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)
	mockLogger.On("Info", mock.Anything).Return()
	mockLogger.On("Error", mock.Anything).Return().Maybe()
	mockLogger.On("WithErr", mock.AnythingOfType("error")).Return(mockLogger).Maybe()

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "fs_test_*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test file structure
	testFiles := map[string]string{
		"file1.txt":        "Hello World",
		"file2.go":         "package main\nfunc main() {}",
		"subdir/file3.txt": "Test content",
		"subdir/file4.go":  "package sub\nfunc Test() {}",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tempDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0644))
	}

	fs := NewFileSystem(mockLogger, FileSystemConfig{
		AllowedDirectory: tempDir,
		BlockedPatterns:  []string{"*.exe", "*.dll"},
	})

	tests := []struct {
		name      string
		input     map[string]interface{}
		wantErr   bool
		checkFunc func(t *testing.T, result mcp.CallToolResult)
	}{
		{
			name: "search by file pattern",
			input: map[string]interface{}{
				"operation": "search",
				"path":      tempDir,
				"pattern":   "*.txt",
				"recursive": true,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result mcp.CallToolResult) {
				assert.Contains(t, result.Content[0].Text, "file1.txt")
				assert.Contains(t, result.Content[0].Text, "file3.txt")
				assert.NotContains(t, result.Content[0].Text, "file2.go")
			},
		},
		{
			name: "search by content",
			input: map[string]interface{}{
				"operation": "search",
				"path":      tempDir,
				"content":   "package main",
				"recursive": true,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result mcp.CallToolResult) {
				assert.Contains(t, result.Content[0].Text, "file2.go")
				assert.NotContains(t, result.Content[0].Text, "file4.go")
			},
		},
		{
			name: "search by pattern and content",
			input: map[string]interface{}{
				"operation": "search",
				"path":      tempDir,
				"pattern":   "*.go",
				"content":   "func Test",
				"recursive": true,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result mcp.CallToolResult) {
				assert.Contains(t, result.Content[0].Text, "file4.go")
				assert.NotContains(t, result.Content[0].Text, "file2.go")
			},
		},
		{
			name: "non-recursive search",
			input: map[string]interface{}{
				"operation": "search",
				"path":      tempDir,
				"pattern":   "*.txt",
				"recursive": false,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result mcp.CallToolResult) {
				assert.Contains(t, result.Content[0].Text, "file1.txt")
				assert.NotContains(t, result.Content[0].Text, "file3.txt")
			},
		},
		{
			name: "search with no matches",
			input: map[string]interface{}{
				"operation": "search",
				"path":      tempDir,
				"content":   "nonexistent content",
				"recursive": true,
			},
			wantErr: false,
			checkFunc: func(t *testing.T, result mcp.CallToolResult) {
				assert.Equal(t, "No matches found", result.Content[0].Text)
			},
		},
		{
			name: "search in non-existent directory",
			input: map[string]interface{}{
				"operation": "search",
				"path":      filepath.Join(tempDir, "nonexistent"),
				"pattern":   "*.txt",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args, err := json.Marshal(tt.input)
			require.NoError(t, err)

			result, err := fs.FileSystemAllInOneTool().Handler(context.Background(), mcp.CallToolParams{
				Name:      FileSystemToolName,
				Arguments: args,
			})

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}

	mockLogger.AssertExpectations(t)
}
