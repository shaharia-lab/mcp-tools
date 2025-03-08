package mcptools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGitHubTest(t *testing.T) (*GitHub, *httptest.Server, func()) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	client := github.NewClient(nil)
	baseUrl, err := url.Parse(server.URL)
	assert.NoError(t, err)

	client.BaseURL = baseUrl.JoinPath("/")

	gh := &GitHub{
		client: client,
		logger: &MockLogger{},
	}

	return gh, server, func() {
		server.Close()
	}
}

func TestGetIssuesTool(t *testing.T) {
	gh := &GitHub{
		client: github.NewClient(nil),
		logger: &MockLogger{},
	}

	tool := gh.GetIssuesTool()

	assert.Equal(t, GitHubIssuesToolName, tool.Name)
	assert.NotEmpty(t, tool.Description)
	assert.NotNil(t, tool.Handler)
	assert.NotEmpty(t, tool.InputSchema)

	// Validate schema structure
	var schema map[string]interface{}
	err := json.Unmarshal(tool.InputSchema, &schema)
	require.NoError(t, err)

	properties, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	// Check required fields
	required, ok := schema["required"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, required, "operation")
	assert.Contains(t, required, "owner")
	assert.Contains(t, required, "repo")

	// Check operation property
	operation, ok := properties["operation"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "string", operation["type"])

	enum, ok := operation["enum"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, enum, "create")
	assert.Contains(t, enum, "get")
	assert.Contains(t, enum, "list")
	assert.Contains(t, enum, "update")
	assert.Contains(t, enum, "comment")
	assert.Contains(t, enum, "close")
}

func TestHandleIssuesOperation_Create(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/issues", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var issueReq github.IssueRequest
		err := json.NewDecoder(r.Body).Decode(&issueReq)
		assert.NoError(t, err)

		assert.Equal(t, "Test Issue", *issueReq.Title)
		assert.Equal(t, "Test Body", *issueReq.Body)
		assert.Equal(t, []string{"bug"}, *issueReq.Labels)
		assert.Equal(t, []string{"testuser"}, *issueReq.Assignees)

		issue := &github.Issue{
			Number: github.Int(1),
			Title:  github.String("Test Issue"),
			Body:   github.String("Test Body"),
		}
		err = json.NewEncoder(w).Encode(issue)
		assert.NoError(t, err)
	})

	input := map[string]interface{}{
		"operation": "create",
		"owner":     "test-owner",
		"repo":      "test-repo",
		"title":     "Test Issue",
		"body":      "Test Body",
		"labels":    []string{"bug"},
		"assignees": []string{"testuser"},
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleIssuesOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubIssuesToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)
	assert.Equal(t, "json", result.Content[0].Type)

	var responseIssue github.Issue
	err = json.Unmarshal([]byte(result.Content[0].Text), &responseIssue)
	require.NoError(t, err)
	assert.Equal(t, 1, *responseIssue.Number)
	assert.Equal(t, "Test Issue", *responseIssue.Title)
}

func TestHandleIssuesOperation_Get(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		issue := &github.Issue{
			Number: github.Int(1),
			Title:  github.String("Test Issue"),
			State:  github.String("open"),
		}
		err := json.NewEncoder(w).Encode(issue)
		assert.NoError(t, err)
	})

	input := map[string]interface{}{
		"operation": "get",
		"owner":     "test-owner",
		"repo":      "test-repo",
		"number":    1,
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleIssuesOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubIssuesToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var responseIssue github.Issue
	err = json.Unmarshal([]byte(result.Content[0].Text), &responseIssue)
	require.NoError(t, err)
	assert.Equal(t, 1, *responseIssue.Number)
	assert.Equal(t, "open", *responseIssue.State)
}

func TestHandleIssuesOperation_List(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/issues", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		issues := []*github.Issue{
			{
				Number: github.Int(1),
				Title:  github.String("Issue 1"),
			},
			{
				Number: github.Int(2),
				Title:  github.String("Issue 2"),
			},
		}
		err := json.NewEncoder(w).Encode(issues)
		assert.NoError(t, err)
	})

	input := map[string]interface{}{
		"operation": "list",
		"owner":     "test-owner",
		"repo":      "test-repo",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleIssuesOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubIssuesToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var issues []*github.Issue
	err = json.Unmarshal([]byte(result.Content[0].Text), &issues)
	require.NoError(t, err)
	assert.Len(t, issues, 2)
	assert.Equal(t, 1, *issues[0].Number)
	assert.Equal(t, 2, *issues[1].Number)
}

func TestHandleIssuesOperation_InvalidOperation(t *testing.T) {
	gh := &GitHub{
		client: github.NewClient(nil),
		logger: &MockLogger{},
	}

	input := map[string]interface{}{
		"operation": "invalid_op",
		"owner":     "test-owner",
		"repo":      "test-repo",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	_, err = gh.handleIssuesOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubIssuesToolName,
		Arguments: inputBytes,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operation")
}

func TestHandleIssuesOperation_InvalidInput(t *testing.T) {
	gh := &GitHub{
		client: github.NewClient(nil),
		logger: &MockLogger{},
	}

	// Invalid JSON input
	_, err := gh.handleIssuesOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubIssuesToolName,
		Arguments: []byte("invalid json"),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal input")
}

func TestHandleIssuesOperation_Close(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/issues/1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)

		var issueReq map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&issueReq)
		assert.NoError(t, err)
		assert.Equal(t, "closed", issueReq["state"])

		issue := &github.Issue{
			Number: github.Int(1),
			State:  github.String("closed"),
		}
		err = json.NewEncoder(w).Encode(issue)
		assert.NoError(t, err)
	})

	input := map[string]interface{}{
		"operation": "close",
		"owner":     "test-owner",
		"repo":      "test-repo",
		"number":    1,
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleIssuesOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubIssuesToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var responseIssue github.Issue
	err = json.Unmarshal([]byte(result.Content[0].Text), &responseIssue)
	require.NoError(t, err)
	assert.Equal(t, "closed", *responseIssue.State)
}
