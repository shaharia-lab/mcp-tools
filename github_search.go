package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
)

// GetSearchTool returns a tool for GitHub search operations
func (g *GitHub) GetSearchTool() mcp.Tool {
	return mcp.Tool{
		Name:        GitHubSearchToolName,
		Description: "Performs GitHub search operations across repositories, code, issues, and users",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"enum": ["repositories", "code", "issues", "users"],
					"description": "Search type to perform"
				},
				"query": {
					"type": "string",
					"description": "Search query"
				},
				"language": {
					"type": "string",
					"description": "Programming language filter for code search"
				},
				"sort": {
					"type": "string",
					"enum": ["stars", "forks", "updated", "best-match"],
					"description": "Sort order for results"
				},
				"order": {
					"type": "string",
					"enum": ["asc", "desc"],
					"description": "Sort direction"
				}
			},
			"required": ["operation", "query"]
		}`),
		Handler: g.handleSearchOperation,
	}
}

func (g *GitHub) handleSearchOperation(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
	ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
	defer span.End()

	var input struct {
		Operation string `json:"operation"`
		Query     string `json:"query"`
		Language  string `json:"language"`
		Sort      string `json:"sort"`
		Order     string `json:"order"`
	}

	g.logger.WithFields(map[string]interface{}{
		"tool":          params.Name,
		"tool_argument": string(params.Arguments),
	}).Info("Received input")

	if err := json.Unmarshal(params.Arguments, &input); err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	var result interface{}
	var err error

	searchOpts := &github.SearchOptions{
		Sort:  input.Sort,
		Order: input.Order,
	}

	g.logger.WithFields(map[string]interface{}{
		"tool":      params.Name,
		"operation": input.Operation,
		"query":     input.Query,
		"language":  input.Language,
		"sort":      input.Sort,
		"order":     input.Order,
	}).Info("Handling search operation")

	// Append language to query if specified
	if input.Language != "" && input.Operation == "code" {
		input.Query = input.Query + " language:" + input.Language
	}

	switch input.Operation {
	case "repositories":
		result, _, err = g.client.Search.Repositories(ctx, input.Query, searchOpts)
	case "code":
		result, _, err = g.client.Search.Code(ctx, input.Query, searchOpts)
	case "issues":
		result, _, err = g.client.Search.Issues(ctx, input.Query, searchOpts)
	case "users":
		result, _, err = g.client.Search.Users(ctx, input.Query, searchOpts)
	default:
		return returnErrorOutput(fmt.Errorf("unsupported operation: %s", input.Operation)), nil
	}

	if err != nil {
		g.logger.WithFields(map[string]interface{}{
			"operation": input.Operation,
			"error":     err,
		}).Error("GitHub search operation failed")

		return returnErrorOutput(err), nil
	}

	m := mustMarshal(result)
	g.logger.WithFields(map[string]interface{}{
		"tool":          GitHubSearchToolName,
		"operation":     input.Operation,
		"result_length": len(m),
	}).Info("GitHub search operation completed successfully")

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "json",
			Text: m,
		}},
	}, nil
}
