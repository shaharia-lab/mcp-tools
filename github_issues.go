package mcptools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai"
)

// GetIssuesTool returns a tool for managing GitHub issues
func (g *GitHub) GetIssuesTool() goai.Tool {
	return goai.Tool{
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

func (g *GitHub) handleIssuesOperation(ctx context.Context, params goai.CallToolParams) (goai.CallToolResult, error) {
	ctx, span := goai.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
	defer span.End()

	g.logger.WithFields(map[string]interface{}{
		"tool_name": params.Name,
		"operation": params.Arguments,
	}).Info("handling issues operation")

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
		return goai.CallToolResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
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
		return returnErrorOutput(fmt.Errorf("unsupported operation: %s", input.Operation)), nil
	}

	if err != nil {
		g.logger.WithFields(map[string]interface{}{
			"tool":                      params.Name,
			goai.ErrorLogField: err,
			"operation":                 input.Operation,
		}).Error("GitHub issues operation failed")

		return returnErrorOutput(err), nil
	}

	marshalledResult := mustMarshal(result)

	g.logger.WithFields(map[string]interface{}{
		"tool":          params.Name,
		"operation":     input.Operation,
		"result_length": len(marshalledResult),
	}).Info("GitHub issues operation completed successfully")

	return goai.CallToolResult{
		Content: []goai.ToolResultContent{{
			Type: "json",
			Text: marshalledResult,
		}},
	}, nil
}
