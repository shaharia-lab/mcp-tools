package mcptools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/shaharia-lab/goai"
	"github.com/stretchr/testify/mock"

	"github.com/google/go-github/v60/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPullRequestsTool(t *testing.T) {
	gh := &GitHub{
		client: github.NewClient(nil),
		logger: &MockLogger{},
	}

	tool := gh.GetPullRequestsTool()

	assert.Equal(t, GitHubPullRequestsToolName, tool.Name)
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

	// Validate operation enum values
	operation, ok := properties["operation"].(map[string]interface{})
	require.True(t, ok)
	enum, ok := operation["enum"].([]interface{})
	require.True(t, ok)
	assert.Contains(t, enum, "create")
	assert.Contains(t, enum, "get")
	assert.Contains(t, enum, "list")
	assert.Contains(t, enum, "update")
	assert.Contains(t, enum, "merge")
	assert.Contains(t, enum, "review")
	assert.Contains(t, enum, "list_files")
}

func TestHandlePullRequestsOperation_Create(t *testing.T) {
	// Create mock logger and set up expected calls
	mockLogger := &MockLogger{}

	// Set up the WithFields expectation
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)

	// Set up Info call expectations with the exact messages
	mockLogger.On("Info", []interface{}{"handling pull requests operation"}).Return()
	mockLogger.On("Info", []interface{}{"GitHub pull request operation completed successfully"}).Return() // Fixed message

	gh, server, cleanup := setupGitHubTest(t)
	// Replace the default logger with our configured mock
	gh.logger = mockLogger
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var prReq github.NewPullRequest
		err := json.NewDecoder(r.Body).Decode(&prReq)
		assert.NoError(t, err)

		assert.Equal(t, "Test PR", *prReq.Title)
		assert.Equal(t, "Test Description", *prReq.Body)
		assert.Equal(t, "feature-branch", *prReq.Head)
		assert.Equal(t, "main", *prReq.Base)

		pr := &github.PullRequest{
			Number: github.Int(1),
			Title:  github.String("Test PR"),
			Head:   &github.PullRequestBranch{Ref: github.String("feature-branch")},
			Base:   &github.PullRequestBranch{Ref: github.String("main")},
		}
		json.NewEncoder(w).Encode(pr) // nolint
	})

	input := map[string]interface{}{
		"operation": "create",
		"owner":     "test-owner",
		"repo":      "test-repo",
		"title":     "Test PR",
		"body":      "Test Description",
		"head":      "feature-branch",
		"base":      "main",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handlePullRequestsOperation(context.Background(), goai.CallToolParams{
		Name:      GitHubPullRequestsToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var responsePR github.PullRequest
	err = json.Unmarshal([]byte(result.Content[0].Text), &responsePR)
	require.NoError(t, err)
	assert.Equal(t, 1, *responsePR.Number)
	assert.Equal(t, "Test PR", *responsePR.Title)
}

func TestHandlePullRequestsOperation_Get(t *testing.T) {
	// Create mock logger and set up expected calls
	mockLogger := &MockLogger{}

	// Set up the WithFields expectation
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)

	// Set up Info call expectations with the exact messages
	mockLogger.On("Info", []interface{}{"handling pull requests operation"}).Return()
	mockLogger.On("Info", []interface{}{"GitHub pull request operation completed successfully"}).Return()

	gh, server, cleanup := setupGitHubTest(t)
	// Replace the default logger with our configured mock
	gh.logger = mockLogger
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/pulls/1", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		pr := &github.PullRequest{
			Number: github.Int(1),
			Title:  github.String("Test PR"),
			State:  github.String("open"),
		}
		err := json.NewEncoder(w).Encode(pr)
		require.NoError(t, err)
	})

	input := map[string]interface{}{
		"operation": "get",
		"owner":     "test-owner",
		"repo":      "test-repo",
		"number":    1,
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handlePullRequestsOperation(context.Background(), goai.CallToolParams{
		Name:      GitHubPullRequestsToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var pr github.PullRequest
	err = json.Unmarshal([]byte(result.Content[0].Text), &pr)
	require.NoError(t, err)
	assert.Equal(t, 1, *pr.Number)
	assert.Equal(t, "open", *pr.State)
}

func TestHandlePullRequestsOperation_Review(t *testing.T) {
	// Create mock logger and set up expected calls
	mockLogger := &MockLogger{}

	// Set up the WithFields expectation
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)

	// Set up Info call expectations with the exact messages
	mockLogger.On("Info", []interface{}{"handling pull requests operation"}).Return()
	mockLogger.On("Info", []interface{}{"GitHub pull request operation completed successfully"}).Return()

	gh, server, cleanup := setupGitHubTest(t)
	// Replace the default logger with our configured mock
	gh.logger = mockLogger
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/pulls/1/reviews", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var reviewReq github.PullRequestReviewRequest
		err := json.NewDecoder(r.Body).Decode(&reviewReq)
		assert.NoError(t, err)

		assert.Equal(t, "LGTM!", *reviewReq.Body)
		assert.Equal(t, "APPROVE", *reviewReq.Event)

		review := &github.PullRequestReview{
			ID:    github.Int64(1),
			State: github.String("APPROVED"),
			Body:  github.String("LGTM!"),
		}
		err = json.NewEncoder(w).Encode(review)
		assert.NoError(t, err)
	})

	input := map[string]interface{}{
		"operation":      "review",
		"owner":          "test-owner",
		"repo":           "test-repo",
		"number":         1,
		"review_comment": "LGTM!",
		"review_event":   "APPROVE",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handlePullRequestsOperation(context.Background(), goai.CallToolParams{
		Name:      GitHubPullRequestsToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var review github.PullRequestReview
	err = json.Unmarshal([]byte(result.Content[0].Text), &review)
	require.NoError(t, err)
	assert.Equal(t, "APPROVED", *review.State)
	assert.Equal(t, "LGTM!", *review.Body)
}

func TestHandlePullRequestsOperation_Merge(t *testing.T) {
	// Create mock logger and set up expected calls
	mockLogger := &MockLogger{}

	// Set up the WithFields expectation
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)

	// Set up Info call expectations with the exact messages
	mockLogger.On("Info", []interface{}{"handling pull requests operation"}).Return()
	mockLogger.On("Info", []interface{}{"GitHub pull request operation completed successfully"}).Return()

	gh, server, cleanup := setupGitHubTest(t)
	// Replace the default logger with our configured mock
	gh.logger = mockLogger
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/pulls/1/merge", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)

		result := &github.PullRequestMergeResult{
			Merged:  github.Bool(true),
			Message: github.String("Pull Request successfully merged"),
		}
		err := json.NewEncoder(w).Encode(result)
		assert.NoError(t, err)
	})

	input := map[string]interface{}{
		"operation": "merge",
		"owner":     "test-owner",
		"repo":      "test-repo",
		"number":    1,
		"body":      "Merging feature into main",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handlePullRequestsOperation(context.Background(), goai.CallToolParams{
		Name:      GitHubPullRequestsToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var mergeResult github.PullRequestMergeResult
	err = json.Unmarshal([]byte(result.Content[0].Text), &mergeResult)
	require.NoError(t, err)
	assert.True(t, *mergeResult.Merged)
}

func TestHandlePullRequestsOperation_ListFiles(t *testing.T) {
	// Create mock logger and set up expected calls
	mockLogger := &MockLogger{}

	// Set up the WithFields expectation
	mockLogger.On("WithFields", mock.Anything).Return(mockLogger)

	// Set up Info call expectations with the exact messages
	mockLogger.On("Info", []interface{}{"handling pull requests operation"}).Return()
	mockLogger.On("Info", []interface{}{"GitHub pull request operation completed successfully"}).Return()

	gh, server, cleanup := setupGitHubTest(t)
	// Replace the default logger with our configured mock
	gh.logger = mockLogger
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/pulls/1/files", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		files := []*github.CommitFile{
			{
				Filename: github.String("file1.go"),
				Status:   github.String("modified"),
			},
			{
				Filename: github.String("file2.go"),
				Status:   github.String("added"),
			},
		}
		err := json.NewEncoder(w).Encode(files)
		assert.NoError(t, err)
	})

	input := map[string]interface{}{
		"operation": "list_files",
		"owner":     "test-owner",
		"repo":      "test-repo",
		"number":    1,
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handlePullRequestsOperation(context.Background(), goai.CallToolParams{
		Name:      GitHubPullRequestsToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var files []*github.CommitFile
	err = json.Unmarshal([]byte(result.Content[0].Text), &files)
	require.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "file1.go", *files[0].Filename)
	assert.Equal(t, "modified", *files[0].Status)
}
