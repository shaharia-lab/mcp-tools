package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"go.opentelemetry.io/otel/attribute"
)

const FileSystemToolName = "filesystem"

// FileSystem represents a wrapper around filesystem operations
type FileSystem struct {
	logger observability.Logger
	config FileSystemConfig
}

// FileSystemConfig holds the configuration for the FileSystem tool
type FileSystemConfig struct {
	AllowedDirectory string   // Base directory for all operations
	BlockedPatterns  []string // Patterns to block (e.g., "*.exe", "*.dll")
}

// NewFileSystem creates a new instance of FileSystem
func NewFileSystem(logger observability.Logger, config FileSystemConfig) *FileSystem {
	return &FileSystem{
		logger: logger,
		config: config,
	}
}

// FileSystemAllInOneTool returns a Tool that performs filesystem operations
func (fs *FileSystem) FileSystemAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        FileSystemToolName,
		Description: "Performs filesystem operations like list, read, write, create, delete files and directories",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"enum": ["list", "tree", "read", "write", "create", "delete", "mkdir", "search"],
					"description": "Filesystem operation to perform"
				},
				"path": {
					"type": "string",
					"description": "Target path for the operation"
				},
				"content": {
					"type": "string",
					"description": "Content for write operations"
				},
				"recursive": {
					"type": "boolean",
					"description": "Whether to perform operation recursively",
					"default": false
				},
				"minutes": {
					"type": "integer",
					"description": "Find files modified within last N minutes"
				},
				"pattern": {
					"type": "string",
					"description": "File name pattern to match (e.g., *.txt)"
				}
			},
			"required": ["operation", "path"]
		}`),

		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
			span.SetAttributes(
				attribute.String("tool_name", params.Name),
				attribute.String("tool_argument", string(params.Arguments)),
			)
			defer span.End()

			fs.logger.WithFields(map[string]interface{}{
				"tool":      params.Name,
				"arguments": string(params.Arguments),
			}).Info("Starting filesystem operation")

			var input struct {
				Operation string `json:"operation"`
				Path      string `json:"path"`
				Content   string `json:"content"`
				Recursive bool   `json:"recursive"`
				Pattern   string `json:"pattern"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				fs.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"raw_input":                 string(params.Arguments),
				}).Error("Failed to unmarshal input parameters")

				span.RecordError(err)
				return returnErrorOutput(err), nil
			}

			// Validate path is within allowed directory
			absPath, err := filepath.Abs(input.Path)
			if err != nil {
				fs.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"path":                      input.Path,
				}).Error("Failed to resolve absolute path")

				span.RecordError(err)
				return returnErrorOutput(err), nil
			}

			if !fs.isPathAllowed(absPath) {
				err = fmt.Errorf("path outside allowed directory: %s", input.Path)
				fs.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"path":                      input.Path,
					"allowed_directory":         fs.config.AllowedDirectory,
				}).Error("Access denied")

				span.RecordError(err)
				return returnErrorOutput(err), nil
			}

			var result mcp.CallToolResult
			var opErr error

			fs.logger.WithFields(map[string]interface{}{
				"tool":      FileSystemToolName,
				"operation": input.Operation,
				"path":      input.Path,
			}).Info("Executing filesystem operation")

			switch input.Operation {
			case "list":
				result, opErr = fs.handleList(ctx, absPath, input.Recursive)
			case "tree":
				result, opErr = fs.handleTree(absPath)
			case "read":
				result, opErr = fs.handleRead(absPath)
			case "write":
				result, opErr = fs.handleWrite(absPath, input.Content)
			case "create":
				result, opErr = fs.handleCreate(absPath)
			case "delete":
				result, opErr = fs.handleDelete(absPath, input.Recursive)
			case "mkdir":
				result, opErr = fs.handleMkdir(absPath)
			case "search":
				result, opErr = fs.handleSearch(absPath, input.Pattern, input.Content, input.Recursive)
			default:
				opErr = fmt.Errorf("unsupported operation: %s", input.Operation)
			}

			if opErr != nil {
				fs.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: opErr,
					"operation":                 input.Operation,
					"path":                      input.Path,
					"tool":                      FileSystemToolName,
				}).Error("Operation failed")

				span.RecordError(opErr)
				return returnErrorOutput(opErr), nil
			}

			fs.logger.WithFields(map[string]interface{}{
				"tool":      FileSystemToolName,
				"operation": input.Operation,
				"path":      input.Path,
			}).Info("Operation completed successfully")

			return result, nil
		},
	}
}

func (fs *FileSystem) handleList(ctx context.Context, path string, recursive bool) (mcp.CallToolResult, error) {
	if err := fs.validatePath(path); err != nil {
		return mcp.CallToolResult{}, err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to read directory: %w", err)
	}

	type fileInfo struct {
		Name    string `json:"name"`
		IsDir   bool   `json:"is_dir"`
		Size    int64  `json:"size"`
		ModTime string `json:"mod_time"`
	}

	var files []fileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, fileInfo{
			Name:    entry.Name(),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().Format(time.RFC3339),
		})

		if recursive && entry.IsDir() {
			subPath := filepath.Join(path, entry.Name())
			subResult, err := fs.handleList(ctx, subPath, recursive)
			if err != nil {
				fs.logger.WithFields(map[string]interface{}{
					"error": err,
					"path":  subPath,
				}).Warn("Failed to list subdirectory")
				continue
			}
			// Append subdirectory results
			var subFiles []fileInfo
			if err := json.Unmarshal([]byte(subResult.Content[0].Text), &subFiles); err == nil {
				files = append(files, subFiles...)
			}
		}
	}

	resultJSON, err := json.MarshalIndent(files, "", "  ")
	if err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to marshal result: %w", err)
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: string(resultJSON),
		}},
	}, nil
}

func (fs *FileSystem) handleTree(path string) (mcp.CallToolResult, error) {
	if err := fs.validatePath(path); err != nil {
		return mcp.CallToolResult{}, err
	}

	var result strings.Builder

	var walk func(string, string, int) error
	walk = func(p string, prefix string, depth int) error {
		if err := fs.validatePath(p); err != nil {
			return err
		}

		entries, err := os.ReadDir(p)
		if err != nil {
			return err
		}

		for i, entry := range entries {
			isLast := i == len(entries)-1
			connector := "├──"
			if isLast {
				connector = "└──"
			}

			result.WriteString(fmt.Sprintf("%s%s %s\n", prefix, connector, entry.Name()))

			if entry.IsDir() {
				newPrefix := prefix
				if isLast {
					newPrefix += "    "
				} else {
					newPrefix += "│   "
				}
				if err := walk(filepath.Join(p, entry.Name()), newPrefix, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := walk(path, "", 0); err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to generate tree: %w", err)
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: result.String(),
		}},
	}, nil
}

func (fs *FileSystem) handleRead(path string) (mcp.CallToolResult, error) {
	if err := fs.validatePath(path); err != nil {
		return mcp.CallToolResult{}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to read file: %w", err)
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: string(content),
		}},
	}, nil
}

func (fs *FileSystem) handleWrite(path string, content string) (mcp.CallToolResult, error) {
	if err := fs.validatePath(path); err != nil {
		return mcp.CallToolResult{}, err
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to write file: %w", err)
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path),
		}},
	}, nil
}

func (fs *FileSystem) handleCreate(path string) (mcp.CallToolResult, error) {
	if err := fs.validatePath(path); err != nil {
		return mcp.CallToolResult{}, err
	}

	file, err := os.Create(path)
	if err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully created file: %s", path),
		}},
	}, nil
}

func (fs *FileSystem) handleDelete(path string, recursive bool) (mcp.CallToolResult, error) {
	if err := fs.validatePath(path); err != nil {
		return mcp.CallToolResult{}, err
	}

	var err error
	if recursive {
		// For recursive delete, we need to validate the entire subtree
		err = filepath.Walk(path, func(subPath string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			return fs.validatePath(subPath)
		})
		if err != nil {
			return mcp.CallToolResult{}, fmt.Errorf("validation failed for recursive delete: %w", err)
		}
		err = os.RemoveAll(path)
	} else {
		err = os.Remove(path)
	}

	if err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to delete: %w", err)
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully deleted: %s", path),
		}},
	}, nil
}

func (fs *FileSystem) handleMkdir(path string) (mcp.CallToolResult, error) {
	if err := fs.validatePath(path); err != nil {
		return mcp.CallToolResult{}, err
	}

	if err := os.MkdirAll(path, 0755); err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to create directory: %w", err)
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "text",
			Text: fmt.Sprintf("Successfully created directory: %s", path),
		}},
	}, nil
}

// isPathAllowed checks if the given path is within the allowed directory
func (fs *FileSystem) isPathAllowed(path string) bool {
	if fs.config.AllowedDirectory == "" {
		return true
	}

	allowedAbs, err := filepath.Abs(fs.config.AllowedDirectory)
	if err != nil {
		fs.logger.WithFields(map[string]interface{}{
			observability.ErrorLogField: err,
			"allowed_directory":         fs.config.AllowedDirectory,
		}).Error("Failed to resolve allowed directory path")
		return false
	}

	// Clean and standardize paths
	path = filepath.Clean(path)
	allowedAbs = filepath.Clean(allowedAbs)

	// Check if the path is within allowed directory
	rel, err := filepath.Rel(allowedAbs, path)
	if err != nil {
		return false
	}

	// Check if the path doesn't start with ".." which would indicate
	// it's outside the allowed directory
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

// isPathBlocked checks if the path matches any blocked patterns
func (fs *FileSystem) isPathBlocked(path string) bool {
	for _, pattern := range fs.config.BlockedPatterns {
		matched, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			fs.logger.WithFields(map[string]interface{}{
				observability.ErrorLogField: err,
				"pattern":                   pattern,
				"path":                      path,
			}).Error("Failed to match pattern")
			continue
		}
		if matched {
			return true
		}
	}
	return false
}

// validatePath combines path validation checks
func (fs *FileSystem) validatePath(path string) error {
	if !fs.isPathAllowed(path) {
		return fmt.Errorf("path is outside allowed directory: %s", path)
	}
	if fs.isPathBlocked(path) {
		return fmt.Errorf("path matches blocked pattern: %s", path)
	}
	return nil
}

func (fs *FileSystem) handleSearch(root string, pattern string, searchContent string, recursive bool) (mcp.CallToolResult, error) {
	if err := fs.validatePath(root); err != nil {
		return mcp.CallToolResult{}, err
	}

	var matches []string
	searchContent = strings.TrimSpace(searchContent)

	walkFn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Handle directory traversal
		if info.IsDir() {
			// Skip subdirectories if not recursive
			if !recursive && path != root {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if we should process this file
		shouldProcess := true

		// Check file pattern match if specified
		if pattern != "" {
			matched, err := filepath.Match(pattern, filepath.Base(path))
			if err != nil {
				return err
			}
			shouldProcess = shouldProcess && matched
		}

		// Check content if specified
		if shouldProcess && searchContent != "" {
			data, err := os.ReadFile(path)
			if err != nil {
				fs.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"path":                      path,
				}).Error("Failed to read file for content search")
				return nil // Skip files we can't read
			}
			shouldProcess = strings.Contains(string(data), searchContent)
		}

		// Add to matches if all conditions are met
		if shouldProcess {
			relPath, err := filepath.Rel(fs.config.AllowedDirectory, path)
			if err != nil {
				return err
			}
			matches = append(matches, relPath)
		}

		return nil
	}

	err := filepath.Walk(root, walkFn)
	if err != nil {
		fs.logger.WithFields(map[string]interface{}{
			observability.ErrorLogField: err,
			"root":                      root,
			"pattern":                   pattern,
			"content":                   searchContent,
		}).Error("Failed to search files")
		return mcp.CallToolResult{}, err
	}

	if len(matches) == 0 {
		return mcp.CallToolResult{
			Content: []mcp.ToolResultContent{
				{
					Type: "text",
					Text: "No matches found",
				},
			},
		}, nil
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{
			{
				Type: "text",
				Text: strings.Join(matches, "\n"),
			},
		},
	}, nil
}
