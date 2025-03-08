package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai/mcp"
)

func TestHandleSearchOperation(t *testing.T) {
	gh, server, cleanup := setupGitHubTest(t)
	defer cleanup()

	mux := http.NewServeMux()
	server.Config.Handler = mux

	tests := []struct {
		name          string
		input         map[string]interface{}
		mockPath      string
		mockResponse  interface{}
		expectedError bool
	}{
		{
			name: "search repositories",
			input: map[string]interface{}{
				"operation": "repositories",
				"query":     "golang",
				"sort":      "stars",
				"order":     "desc",
			},
			mockPath: "/search/repositories",
			mockResponse: &github.RepositoriesSearchResult{
				Total:             github.Int(1),
				IncompleteResults: github.Bool(false),
				Repositories: []*github.Repository{
					{
						Name:            github.String("test-repo"),
						FullName:        github.String("test-owner/test-repo"),
						Description:     github.String("Test repository"),
						StargazersCount: github.Int(100),
					},
				},
			},
		},
		{
			name: "search code",
			input: map[string]interface{}{
				"operation": "code",
				"query":     "fmt.Println",
				"language":  "go",
			},
			mockPath: "/search/code",
			mockResponse: &github.CodeSearchResult{
				Total:             github.Int(1),
				IncompleteResults: github.Bool(false),
				CodeResults: []*github.CodeResult{
					{
						Name:       github.String("main.go"),
						Path:       github.String("src/main.go"),
						Repository: &github.Repository{FullName: github.String("test-owner/test-repo")},
					},
				},
			},
		},
		{
			name: "search issues",
			input: map[string]interface{}{
				"operation": "issues",
				"query":     "is:open label:bug",
			},
			mockPath: "/search/issues",
			mockResponse: &github.IssuesSearchResult{
				Total:             github.Int(1),
				IncompleteResults: github.Bool(false),
				Issues: []*github.Issue{
					{
						Number: github.Int(1),
						Title:  github.String("Test Issue"),
						State:  github.String("open"),
					},
				},
			},
		},
		{
			name: "search users",
			input: map[string]interface{}{
				"operation": "users",
				"query":     "type:user",
			},
			mockPath: "/search/users",
			mockResponse: &github.UsersSearchResult{
				Total:             github.Int(1),
				IncompleteResults: github.Bool(false),
				Users: []*github.User{
					{
						Login: github.String("testuser"),
						Name:  github.String("Test User"),
					},
				},
			},
		},
		{
			name: "invalid operation",
			input: map[string]interface{}{
				"operation": "invalid",
				"query":     "test",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.expectedError {
				mux.HandleFunc(tt.mockPath, func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "GET" {
						t.Errorf("Expected GET request, got %s", r.Method)
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}

					// Verify query parameters
					query := r.URL.Query().Get("q")
					if query == "" {
						t.Error("Query parameter 'q' is missing")
						w.WriteHeader(http.StatusBadRequest)
						return
					}

					// If language was specified for code search, verify it's in the query
					if language, ok := tt.input["language"]; ok && tt.input["operation"] == "code" {
						expectedLanguage := fmt.Sprintf("language:%s", language)
						if !strings.Contains(query, expectedLanguage) {
							t.Errorf("Expected query to contain %s, got %s", expectedLanguage, query)
						}
					}

					w.Header().Set("Content-Type", "application/json")
					err := json.NewEncoder(w).Encode(tt.mockResponse)
					assert.NoError(t, err)
				})
			}

			inputBytes, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("Failed to marshal input: %v", err)
			}

			result, err := gh.handleSearchOperation(context.Background(), mcp.CallToolParams{
				Name:      GitHubSearchToolName,
				Arguments: inputBytes,
			})

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result.Content) == 0 {
				t.Fatal("Expected non-empty result content")
			}

			if result.Content[0].Type != "json" {
				t.Errorf("Expected content type 'json', got %s", result.Content[0].Type)
			}

			// Verify the response can be unmarshaled back into the appropriate type
			switch tt.input["operation"] {
			case "repositories":
				var searchResult github.RepositoriesSearchResult
				if err := json.Unmarshal([]byte(result.Content[0].Text), &searchResult); err != nil {
					t.Errorf("Failed to unmarshal repositories result: %v", err)
				}
			case "code":
				var searchResult github.CodeSearchResult
				if err := json.Unmarshal([]byte(result.Content[0].Text), &searchResult); err != nil {
					t.Errorf("Failed to unmarshal code result: %v", err)
				}
			case "issues":
				var searchResult github.IssuesSearchResult
				if err := json.Unmarshal([]byte(result.Content[0].Text), &searchResult); err != nil {
					t.Errorf("Failed to unmarshal issues result: %v", err)
				}
			case "users":
				var searchResult github.UsersSearchResult
				if err := json.Unmarshal([]byte(result.Content[0].Text), &searchResult); err != nil {
					t.Errorf("Failed to unmarshal users result: %v", err)
				}
			}
		})
	}
}
