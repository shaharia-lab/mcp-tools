package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
)

// GetIssuesTool returns a tool for managing GitHub issues
func (g *GitHub) GetIssuesTool() mcp.Tool {
	return mcp.Tool{
		Name:        GitHubIssuesToolName,
		Description: "Manages GitHub issues - create, list, update, comment",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"enum": ["create", "get", "list", "update", "comment", "close"],
					"description": "Issue operation to perform"
				},
				"owner": {
					"type": "string",
					"description": "Repository owner"
				},
				"repo": {
					"type": "string",
					"description": "Repository name"
				},
				"number": {
					"type": "integer",
					"description": "Issue number"
				},
				"title": {
					"type": "string",
					"description": "Issue title for creation/update"
				},
				"body": {
					"type": "string",
					"description": "Issue body or comment content"
				},
				"labels": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Issue labels"
				},
				"assignees": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Issue assignees"
				}
			},
			"required": ["operation", "owner", "repo"]
		}`),
		Handler: g.handleIssuesOperation,
	}
}

func (g *GitHub) handleIssuesOperation(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
	ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
	defer span.End()

	var input struct {
		Operation string   `json:"operation"`
		Owner     string   `json:"owner"`
		Repo      string   `json:"repo"`
		Number    int      `json:"number"`
		Title     string   `json:"title"`
		Body      string   `json:"body"`
		Labels    []string `json:"labels"`
		Assignees []string `json:"assignees"`
	}

	if err := json.Unmarshal(params.Arguments, &input); err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	var result interface{}
	var err error

	switch input.Operation {
	case "create":
		result, _, err = g.client.Issues.Create(ctx, input.Owner, input.Repo, &github.IssueRequest{
			Title:     &input.Title,
			Body:      &input.Body,
			Labels:    &input.Labels,
			Assignees: &input.Assignees,
		})
	case "get":
		result, _, err = g.client.Issues.Get(ctx, input.Owner, input.Repo, input.Number)
	case "list":
		result, _, err = g.client.Issues.ListByRepo(ctx, input.Owner, input.Repo, &github.IssueListByRepoOptions{})
	case "update":
		result, _, err = g.client.Issues.Edit(ctx, input.Owner, input.Repo, input.Number, &github.IssueRequest{
			Title:     &input.Title,
			Body:      &input.Body,
			Labels:    &input.Labels,
			Assignees: &input.Assignees,
		})
	case "comment":
		result, _, err = g.client.Issues.CreateComment(ctx, input.Owner, input.Repo, input.Number, &github.IssueComment{
			Body: &input.Body,
		})
	case "close":
		state := "closed"
		result, _, err = g.client.Issues.Edit(ctx, input.Owner, input.Repo, input.Number, &github.IssueRequest{
			State: &state,
		})
	default:
		return mcp.CallToolResult{}, fmt.Errorf("unsupported operation: %s", input.Operation)
	}

	if err != nil {
		return mcp.CallToolResult{}, err
	}

	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{
			Type: "json",
			Text: mustMarshal(result),
		}},
	}, nil
}
