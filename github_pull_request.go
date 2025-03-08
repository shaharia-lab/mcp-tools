package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
)

// GetPullRequestsTool returns a tool for managing GitHub pull requests
func (g *GitHub) GetPullRequestsTool() mcp.Tool {
	return mcp.Tool{
		Name:        GitHubPullRequestsToolName,
		Description: "Manages GitHub pull requests - create, review, merge",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"enum": ["create", "get", "list", "update", "merge", "review", "list_files"],
					"description": "Pull request operation to perform"
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
					"description": "Pull request number"
				},
				"title": {
					"type": "string",
					"description": "PR title for creation/update"
				},
				"body": {
					"type": "string",
					"description": "PR description"
				},
				"head": {
					"type": "string",
					"description": "Head branch"
				},
				"base": {
					"type": "string",
					"description": "Base branch"
				},
				"review_comment": {
					"type": "string",
					"description": "Review comment"
				},
				"review_event": {
					"type": "string",
					"enum": ["APPROVE", "REQUEST_CHANGES", "COMMENT"],
					"description": "Review event type"
				}
			},
			"required": ["operation", "owner", "repo"]
		}`),
		Handler: g.handlePullRequestsOperation,
	}
}

func (g *GitHub) handlePullRequestsOperation(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
	ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
	defer span.End()

	var input struct {
		Operation     string `json:"operation"`
		Owner         string `json:"owner"`
		Repo          string `json:"repo"`
		Number        int    `json:"number"`
		Title         string `json:"title"`
		Body          string `json:"body"`
		Head          string `json:"head"`
		Base          string `json:"base"`
		ReviewComment string `json:"review_comment"`
		ReviewEvent   string `json:"review_event"`
	}

	if err := json.Unmarshal(params.Arguments, &input); err != nil {
		return mcp.CallToolResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
	}

	var result interface{}
	var err error

	switch input.Operation {
	case "create":
		result, _, err = g.client.PullRequests.Create(ctx, input.Owner, input.Repo, &github.NewPullRequest{
			Title: &input.Title,
			Body:  &input.Body,
			Head:  &input.Head,
			Base:  &input.Base,
		})
	case "get":
		result, _, err = g.client.PullRequests.Get(ctx, input.Owner, input.Repo, input.Number)
	case "list":
		result, _, err = g.client.PullRequests.List(ctx, input.Owner, input.Repo, &github.PullRequestListOptions{})
	case "update":
		result, _, err = g.client.PullRequests.Edit(ctx, input.Owner, input.Repo, input.Number, &github.PullRequest{
			Title: &input.Title,
			Body:  &input.Body,
		})
	case "merge":
		result, _, err = g.client.PullRequests.Merge(ctx, input.Owner, input.Repo, input.Number, input.Body, &github.PullRequestOptions{})
	case "review":
		result, _, err = g.client.PullRequests.CreateReview(ctx, input.Owner, input.Repo, input.Number, &github.PullRequestReviewRequest{
			Body:  &input.ReviewComment,
			Event: &input.ReviewEvent,
		})
	case "list_files":
		result, _, err = g.client.PullRequests.ListFiles(ctx, input.Owner, input.Repo, input.Number, &github.ListOptions{})
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
