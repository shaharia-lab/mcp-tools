package mcptools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
)

const JiraToolName = "jira"

// JiraClient represents a client for interacting with Jira API
type JiraClient struct {
	logger     observability.Logger
	httpClient *http.Client
	baseURL    string
	username   string
	apiToken   string
}

// NewJiraClient creates a new instance of the JiraClient
func NewJiraClient(logger observability.Logger, baseURL, username, apiToken string) *JiraClient {
	return &JiraClient{
		logger:     logger,
		httpClient: &http.Client{},
		baseURL:    baseURL,
		username:   username,
		apiToken:   apiToken,
	}
}

// JiraTools returns a collection of tools for interacting with Jira
func (j *JiraClient) JiraTools() []mcp.Tool {
	return []mcp.Tool{
		j.CreateIssueTool(),
		j.GetIssueTool(),
		j.SearchIssuesTool(),
		j.UpdateIssueTool(),
		j.AddCommentTool(),
	}
}

// CreateIssueTool returns a tool for creating a new Jira issue
func (j *JiraClient) CreateIssueTool() mcp.Tool {
	return mcp.Tool{
		Name:        JiraToolName + ".create",
		Description: "Create a new Jira issue",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"project": {
					"type": "string",
					"description": "Project key (e.g., 'PROJ')"
				},
				"issueType": {
					"type": "string",
					"description": "Type of issue (e.g., 'Bug', 'Story', 'Task')"
				},
				"summary": {
					"type": "string",
					"description": "Issue summary/title"
				},
				"description": {
					"type": "string",
					"description": "Issue description"
				},
				"priority": {
					"type": "string",
					"description": "Issue priority (e.g., 'High', 'Medium', 'Low')"
				},
				"assignee": {
					"type": "string",
					"description": "Username of the assignee"
				},
				"labels": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "List of labels to apply to the issue"
				},
				"customFields": {
					"type": "object",
					"description": "Custom fields as key-value pairs"
				}
			},
			"required": ["project", "issueType", "summary"]
		}`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			var input struct {
				Project      string                 `json:"project"`
				IssueType    string                 `json:"issueType"`
				Summary      string                 `json:"summary"`
				Description  string                 `json:"description"`
				Priority     string                 `json:"priority"`
				Assignee     string                 `json:"assignee"`
				Labels       []string               `json:"labels"`
				CustomFields map[string]interface{} `json:"customFields"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse input: %v", err)), nil
			}

			// Build the request payload
			payload := map[string]interface{}{
				"fields": map[string]interface{}{
					"project": map[string]string{
						"key": input.Project,
					},
					"issuetype": map[string]string{
						"name": input.IssueType,
					},
					"summary": input.Summary,
				},
			}

			// Add optional fields if provided
			fields := payload["fields"].(map[string]interface{})

			if input.Description != "" {
				fields["description"] = input.Description
			}

			if input.Priority != "" {
				fields["priority"] = map[string]string{
					"name": input.Priority,
				}
			}

			if input.Assignee != "" {
				fields["assignee"] = map[string]string{
					"name": input.Assignee,
				}
			}

			if len(input.Labels) > 0 {
				fields["labels"] = input.Labels
			}

			// Add any custom fields
			for key, value := range input.CustomFields {
				fields[key] = value
			}

			// Make the API call
			data, err := json.Marshal(payload)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
			}

			url := fmt.Sprintf("%s/rest/api/2/issue", j.baseURL)
			resp, err := j.makeRequest(ctx, http.MethodPost, url, bytes.NewBuffer(data))
			if err != nil {
				return j.errorResult(fmt.Sprintf("API request failed: %v", err)), nil
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to read response: %v", err)), nil
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return j.errorResult(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(respBody))), nil
			}

			var result map[string]interface{}
			if err := json.Unmarshal(respBody, &result); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse response: %v", err)), nil
			}

			prettyJSON, _ := json.MarshalIndent(result, "", "  ")
			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{
					Type: "text",
					Text: fmt.Sprintf("Issue created successfully: %s", string(prettyJSON)),
				}},
				IsError: false,
			}, nil
		},
	}
}

// GetIssueTool returns a tool for retrieving a specific Jira issue
func (j *JiraClient) GetIssueTool() mcp.Tool {
	return mcp.Tool{
		Name:        JiraToolName + ".get",
		Description: "Get details of a specific Jira issue",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"issueKey": {
					"type": "string",
					"description": "Issue key (e.g., 'PROJ-123')"
				},
				"fields": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "Optional list of fields to include (default: all)"
				}
			},
			"required": ["issueKey"]
		}`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			var input struct {
				IssueKey string   `json:"issueKey"`
				Fields   []string `json:"fields"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse input: %v", err)), nil
			}

			url := fmt.Sprintf("%s/rest/api/2/issue/%s", j.baseURL, input.IssueKey)

			// Add fields parameter if specified
			if len(input.Fields) > 0 {
				fieldsParam := "fields=" + input.Fields[0]
				for _, field := range input.Fields[1:] {
					fieldsParam += "," + field
				}
				url += "?" + fieldsParam
			}

			resp, err := j.makeRequest(ctx, http.MethodGet, url, nil)
			if err != nil {
				return j.errorResult(fmt.Sprintf("API request failed: %v", err)), nil
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to read response: %v", err)), nil
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return j.errorResult(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(respBody))), nil
			}

			var result map[string]interface{}
			if err := json.Unmarshal(respBody, &result); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse response: %v", err)), nil
			}

			prettyJSON, _ := json.MarshalIndent(result, "", "  ")
			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{Type: "text", Text: string(prettyJSON)}},
				IsError: false,
			}, nil
		},
	}
}

// SearchIssuesTool returns a tool for searching Jira issues
func (j *JiraClient) SearchIssuesTool() mcp.Tool {
	return mcp.Tool{
		Name:        JiraToolName + ".search",
		Description: "Search for Jira issues using JQL",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"jql": {
					"type": "string",
					"description": "JQL query string"
				},
				"startAt": {
					"type": "integer",
					"description": "Index of the first result to return (0-based, default: 0)"
				},
				"maxResults": {
					"type": "integer",
					"description": "Maximum number of results to return (default: 50)"
				},
				"fields": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "List of fields to include in the results for /rest/api/2/search JIRA API endpoint"
				}
			},
			"required": ["jql"]
		}`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			var input struct {
				JQL        string   `json:"jql"`
				StartAt    int      `json:"startAt"`
				MaxResults int      `json:"maxResults"`
				Fields     []string `json:"fields"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse input: %v", err)), nil
			}

			// Set default values if not provided
			if input.MaxResults == 0 {
				input.MaxResults = 5
			}

			// Build the request payload
			payload := map[string]interface{}{
				"jql":        input.JQL,
				"startAt":    input.StartAt,
				"maxResults": input.MaxResults,
			}

			if len(input.Fields) > 0 {
				payload["fields"] = input.Fields
			}

			data, err := json.Marshal(payload)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
			}

			url := fmt.Sprintf("%s/rest/api/2/search", j.baseURL)
			resp, err := j.makeRequest(ctx, http.MethodPost, url, bytes.NewBuffer(data))
			if err != nil {
				return j.errorResult(fmt.Sprintf("API request failed: %v", err)), nil
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to read response: %v", err)), nil
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return j.errorResult(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(respBody))), nil
			}

			var result map[string]interface{}
			if err := json.Unmarshal(respBody, &result); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse response: %v", err)), nil
			}

			prettyJSON, _ := json.MarshalIndent(result, "", "  ")
			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{Type: "text", Text: string(prettyJSON)}},
				IsError: false,
			}, nil
		},
	}
}

// UpdateIssueTool returns a tool for updating an existing Jira issue
func (j *JiraClient) UpdateIssueTool() mcp.Tool {
	return mcp.Tool{
		Name:        JiraToolName + ".update",
		Description: "Update an existing Jira issue",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"issueKey": {
					"type": "string",
					"description": "Issue key (e.g., 'PROJ-123')"
				},
				"summary": {
					"type": "string",
					"description": "Updated issue summary/title"
				},
				"description": {
					"type": "string",
					"description": "Updated issue description"
				},
				"priority": {
					"type": "string",
					"description": "Updated issue priority"
				},
				"assignee": {
					"type": "string",
					"description": "Username of the new assignee"
				},
				"labels": {
					"type": "array",
					"items": {
						"type": "string"
					},
					"description": "Updated list of labels"
				},
				"customFields": {
					"type": "object",
					"description": "Custom fields to update as key-value pairs"
				}
			},
			"required": ["issueKey"]
		}`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			var input struct {
				IssueKey     string                 `json:"issueKey"`
				Summary      string                 `json:"summary"`
				Description  string                 `json:"description"`
				Priority     string                 `json:"priority"`
				Assignee     string                 `json:"assignee"`
				Labels       []string               `json:"labels"`
				CustomFields map[string]interface{} `json:"customFields"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse input: %v", err)), nil
			}

			// Build the request payload with only the fields to update
			fields := make(map[string]interface{})

			if input.Summary != "" {
				fields["summary"] = input.Summary
			}

			if input.Description != "" {
				fields["description"] = input.Description
			}

			if input.Priority != "" {
				fields["priority"] = map[string]string{
					"name": input.Priority,
				}
			}

			if input.Assignee != "" {
				fields["assignee"] = map[string]string{
					"name": input.Assignee,
				}
			}

			if input.Labels != nil {
				fields["labels"] = input.Labels
			}

			// Add any custom fields
			for key, value := range input.CustomFields {
				fields[key] = value
			}

			// If no fields to update, return error
			if len(fields) == 0 {
				return j.errorResult("No fields provided for update"), nil
			}

			payload := map[string]interface{}{
				"fields": fields,
			}

			data, err := json.Marshal(payload)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
			}

			url := fmt.Sprintf("%s/rest/api/2/issue/%s", j.baseURL, input.IssueKey)
			resp, err := j.makeRequest(ctx, http.MethodPut, url, bytes.NewBuffer(data))
			if err != nil {
				return j.errorResult(fmt.Sprintf("API request failed: %v", err)), nil
			}
			defer resp.Body.Close()

			// No content is returned on successful update (204)
			if resp.StatusCode == http.StatusNoContent {
				return mcp.CallToolResult{
					Content: []mcp.ToolResultContent{{
						Type: "text",
						Text: fmt.Sprintf("Issue %s updated successfully", input.IssueKey),
					}},
					IsError: false,
				}, nil
			}

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to read response: %v", err)), nil
			}

			return j.errorResult(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(respBody))), nil
		},
	}
}

// AddCommentTool returns a tool for adding a comment to a Jira issue
func (j *JiraClient) AddCommentTool() mcp.Tool {
	return mcp.Tool{
		Name:        JiraToolName + ".comment",
		Description: "Add a comment to a Jira issue",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"issueKey": {
					"type": "string",
					"description": "Issue key (e.g., 'PROJ-123')"
				},
				"comment": {
					"type": "string",
					"description": "Comment text to add to the issue"
				},
				"visibility": {
					"type": "object",
					"properties": {
						"type": {
							"type": "string",
							"description": "Type of visibility restriction ('group' or 'role')"
						},
						"value": {
							"type": "string",
							"description": "The group name or role name for the visibility restriction"
						}
					},
					"description": "Optional visibility restriction for the comment"
				}
			},
			"required": ["issueKey", "comment"]
		}`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			var input struct {
				IssueKey   string `json:"issueKey"`
				Comment    string `json:"comment"`
				Visibility struct {
					Type  string `json:"type"`
					Value string `json:"value"`
				} `json:"visibility"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse input: %v", err)), nil
			}

			// Build the request payload
			payload := map[string]interface{}{
				"body": input.Comment,
			}

			// Add visibility if provided
			if input.Visibility.Type != "" && input.Visibility.Value != "" {
				payload["visibility"] = map[string]string{
					"type":  input.Visibility.Type,
					"value": input.Visibility.Value,
				}
			}

			data, err := json.Marshal(payload)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to marshal payload: %v", err)), nil
			}

			url := fmt.Sprintf("%s/rest/api/2/issue/%s/comment", j.baseURL, input.IssueKey)
			resp, err := j.makeRequest(ctx, http.MethodPost, url, bytes.NewBuffer(data))
			if err != nil {
				return j.errorResult(fmt.Sprintf("API request failed: %v", err)), nil
			}
			defer resp.Body.Close()

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				return j.errorResult(fmt.Sprintf("Failed to read response: %v", err)), nil
			}

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return j.errorResult(fmt.Sprintf("API error (status %d): %s", resp.StatusCode, string(respBody))), nil
			}

			var result map[string]interface{}
			if err := json.Unmarshal(respBody, &result); err != nil {
				return j.errorResult(fmt.Sprintf("Failed to parse response: %v", err)), nil
			}

			prettyJSON, _ := json.MarshalIndent(result, "", "  ")
			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{
					Type: "text",
					Text: fmt.Sprintf("Comment added successfully: %s", string(prettyJSON)),
				}},
				IsError: false,
			}, nil
		},
	}
}

// Helper method to make authenticated HTTP requests to Jira
func (j *JiraClient) makeRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(j.username, j.apiToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	j.logger.WithFields(map[string]interface{}{
		"tool":   JiraToolName,
		"method": method,
		"url":    url,
	}).Info("Making Jira API request")

	return j.httpClient.Do(req)
}

// Helper method to create an error result
func (j *JiraClient) errorResult(message string) mcp.CallToolResult {
	j.logger.WithFields(map[string]interface{}{"tool": JiraToolName}).Error(message)
	return mcp.CallToolResult{
		Content: []mcp.ToolResultContent{{Type: "text", Text: message}},
		IsError: true,
	}
}
