package mcptools

import (
	"context"
	"encoding/json"

	"github.com/google/go-github/v60/github"
	"github.com/shaharia-lab/goai/observability"
	"golang.org/x/oauth2"
)

const (
	GitHubIssuesToolName       = "github_issues"
	GitHubPullRequestsToolName = "github_pull_requests"
	GitHubRepositoryToolName   = "github_repository"
	GitHubSearchToolName       = "github_search"
)

// GitHub represents a wrapper around GitHub API client
type GitHub struct {
	client *github.Client
	logger observability.Logger
	config GitHubConfig
}

type GitHubConfig struct {
	Token string
}

func NewGitHub(logger observability.Logger, config GitHubConfig) *GitHub {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: config.Token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	return &GitHub{
		client: client,
		logger: logger,
		config: config,
	}
}

// Helper function for JSON marshaling
func mustMarshal(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
