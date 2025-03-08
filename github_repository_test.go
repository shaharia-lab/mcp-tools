package mcptools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRepositoryTool(t *testing.T) {
	gh := &GitHub{
		client: github.NewClient(nil),
		logger: &MockLogger{},
	}

	tool := gh.GetRepositoryTool()

	assert.Equal(t, GitHubRepositoryToolName, tool.Name)
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

	// Validate operation enum values
	operation, ok := properties["operation"].(map[string]interface{})
	require.True(t, ok)
	enum, ok := operation["enum"].([]interface{})
	require.True(t, ok)
	expectedOps := []string{"create", "delete", "update", "fork", "list_branches", "create_branch", "protect_branch"}
	for _, op := range expectedOps {
		assert.Contains(t, enum, op)
	}
}

func TestHandleRepositoryOperation_Create(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/user/repos", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		var repo github.Repository
		json.NewDecoder(r.Body).Decode(&repo)
		assert.Equal(t, "test-repo", *repo.Name)
		assert.Equal(t, "Test Repository", *repo.Description)
		assert.True(t, *repo.Private)

		response := &github.Repository{
			ID:          github.Int64(1),
			Name:        github.String("test-repo"),
			Description: github.String("Test Repository"),
			Private:     github.Bool(true),
		}
		json.NewEncoder(w).Encode(response)
	})

	input := map[string]interface{}{
		"operation":   "create",
		"repo":        "test-repo",
		"description": "Test Repository",
		"private":     true,
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleRepositoryOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubRepositoryToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var repo github.Repository
	err = json.Unmarshal([]byte(result.Content[0].Text), &repo)
	require.NoError(t, err)
	assert.Equal(t, "test-repo", *repo.Name)
	assert.True(t, *repo.Private)
}

func TestHandleRepositoryOperation_Delete(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	})

	input := map[string]interface{}{
		"operation": "delete",
		"owner":     "test-owner",
		"repo":      "test-repo",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleRepositoryOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubRepositoryToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var response map[string]string
	err = json.Unmarshal([]byte(result.Content[0].Text), &response)
	require.NoError(t, err)
	assert.Equal(t, "deleted", response["status"])
}

func TestHandleRepositoryOperation_Fork(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/forks", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)

		response := &github.Repository{
			ID:    github.Int64(2),
			Name:  github.String("test-repo"),
			Fork:  github.Bool(true),
			Owner: &github.User{Login: github.String("forked-owner")},
		}
		json.NewEncoder(w).Encode(response)
	})

	input := map[string]interface{}{
		"operation": "fork",
		"owner":     "test-owner",
		"repo":      "test-repo",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleRepositoryOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubRepositoryToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var repo github.Repository
	err = json.Unmarshal([]byte(result.Content[0].Text), &repo)
	require.NoError(t, err)
	assert.True(t, *repo.Fork)
	assert.Equal(t, "forked-owner", *repo.Owner.Login)
}

func TestHandleRepositoryOperation_ListBranches(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/branches", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)

		branches := []*github.Branch{
			{Name: github.String("main")},
			{Name: github.String("develop")},
		}
		json.NewEncoder(w).Encode(branches)
	})

	input := map[string]interface{}{
		"operation": "list_branches",
		"owner":     "test-owner",
		"repo":      "test-repo",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleRepositoryOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubRepositoryToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var branches []*github.Branch
	err = json.Unmarshal([]byte(result.Content[0].Text), &branches)
	require.NoError(t, err)
	assert.Len(t, branches, 2)
	assert.Equal(t, "main", *branches[0].Name)
	assert.Equal(t, "develop", *branches[1].Name)
}

func TestHandleRepositoryOperation_CreateBranch(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	// Mock source branch ref endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/ref/heads/main", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		ref := &github.Reference{
			Ref: github.String("refs/heads/main"),
			Object: &github.GitObject{
				SHA: github.String("abc123"),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(ref); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	// Mock create ref endpoint
	mux.HandleFunc("/repos/test-owner/test-repo/git/refs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var requestBody struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		}

		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if requestBody.Ref == "" || requestBody.SHA == "" {
			t.Error("Required fields missing in request")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := &github.Reference{
			Ref: github.String(requestBody.Ref),
			Object: &github.GitObject{
				SHA:  github.String(requestBody.SHA),
				Type: github.String("commit"),
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Errorf("Failed to encode response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	input := map[string]interface{}{
		"operation":     "create_branch",
		"owner":         "test-owner",
		"repo":          "test-repo",
		"branch":        "feature",
		"source_branch": "main",
	}

	inputBytes, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	result, err := gh.handleRepositoryOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubRepositoryToolName,
		Arguments: inputBytes,
	})

	if err != nil {
		t.Fatalf("handleRepositoryOperation failed: %v", err)
	}

	if len(result.Content) == 0 {
		t.Fatal("Expected non-empty result content")
	}

	var ref github.Reference
	if err := json.Unmarshal([]byte(result.Content[0].Text), &ref); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}

	if ref.Ref == nil {
		t.Fatal("Ref field is nil in response")
	}

	expectedRef := "refs/heads/feature"
	if *ref.Ref != expectedRef {
		t.Errorf("Expected ref to be '%s', got '%s'", expectedRef, *ref.Ref)
	}

	if ref.Object == nil || ref.Object.SHA == nil {
		t.Fatal("Object or SHA is nil in response")
	}

	expectedSHA := "abc123"
	if *ref.Object.SHA != expectedSHA {
		t.Errorf("Expected SHA to be '%s', got '%s'", expectedSHA, *ref.Object.SHA)
	}
}

func TestHandleRepositoryOperation_ProtectBranch(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	mux.HandleFunc("/repos/test-owner/test-repo/branches/main/protection", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)

		var protection github.ProtectionRequest
		json.NewDecoder(r.Body).Decode(&protection)
		assert.True(t, protection.RequiredStatusChecks.Strict)
		assert.Equal(t, 1, protection.RequiredPullRequestReviews.RequiredApprovingReviewCount)

		response := &github.Protection{
			RequiredStatusChecks: &github.RequiredStatusChecks{
				Strict: true,
			},
			RequiredPullRequestReviews: &github.PullRequestReviewsEnforcement{
				RequiredApprovingReviewCount: 1,
			},
		}
		json.NewEncoder(w).Encode(response)
	})

	input := map[string]interface{}{
		"operation": "protect_branch",
		"owner":     "test-owner",
		"repo":      "test-repo",
		"branch":    "main",
	}

	inputBytes, err := json.Marshal(input)
	require.NoError(t, err)

	result, err := gh.handleRepositoryOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubRepositoryToolName,
		Arguments: inputBytes,
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.Content)

	var protection github.Protection
	err = json.Unmarshal([]byte(result.Content[0].Text), &protection)
	require.NoError(t, err)
	assert.True(t, protection.RequiredStatusChecks.Strict)
	assert.Equal(t, 1, protection.RequiredPullRequestReviews.RequiredApprovingReviewCount)
}

func TestHandleRepositoryOperation_InvalidOperation(t *testing.T) {
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

	_, err = gh.handleRepositoryOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubRepositoryToolName,
		Arguments: inputBytes,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported operation")
}

func TestHandleRepositoryOperation_InvalidInput(t *testing.T) {
	gh := &GitHub{
		client: github.NewClient(nil),
		logger: &MockLogger{},
	}

	_, err := gh.handleRepositoryOperation(context.Background(), mcp.CallToolParams{
		Name:      GitHubRepositoryToolName,
		Arguments: []byte("invalid json"),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal input")
}
