package mcptools

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

// JiraClient represents a client for interacting with Jira
type JiraClient struct {
	BaseURL   string
	AuthToken string
	client    *resty.Client
}

// JiraIssue represents the structure of a Jira issue
type JiraIssue struct {
	Key        string            `json:"key"`
	Fields     map[string]interface{} `json:"fields"`
	Changelog  map[string]interface{} `json:"changelog,omitempty"`
}

// NewJiraClient creates a new Jira client
func NewJiraClient(baseURL, authToken string) *JiraClient {
	client := resty.New()
	client.SetBaseURL(baseURL)
	client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", authToken))
	client.SetHeader("Content-Type", "application/json")

	return &JiraClient{
		BaseURL:   baseURL,
		AuthToken: authToken,
		client:    client,
	}
}

// SearchJiraIssues searches for Jira issues based on a query
func (j *JiraClient) SearchJiraIssues(query string) ([]JiraIssue, error) {
	resp, err := j.client.R().
		SetQueryParam("jql", query).
		SetResult(&struct {
			Issues []JiraIssue `json:"issues"`
		}{}).
		Get("/rest/api/3/search")

	if err != nil {
		return nil, fmt.Errorf("failed to search Jira issues: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Jira search request failed with status %d: %s", 
			resp.StatusCode(), resp.String())
	}

	result := resp.Result().(*struct {
		Issues []JiraIssue `json:"issues"`
	})

	return result.Issues, nil
}

// ReadJiraIssue retrieves details of a specific Jira issue
func (j *JiraClient) ReadJiraIssue(issueKey string) (*JiraIssue, error) {
	resp, err := j.client.R().
		SetPathParams(map[string]string{
			"issueKey": issueKey,
		}).
		SetResult(&JiraIssue{}).
		Get("/rest/api/3/issue/{issueKey}")

	if err != nil {
		return nil, fmt.Errorf("failed to read Jira issue: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Jira issue read failed with status %d: %s", 
			resp.StatusCode(), resp.String())
	}

	issue := resp.Result().(*JiraIssue)
	return issue, nil
}

// CreateJiraIssue creates a new issue in Jira
func (j *JiraClient) CreateJiraIssue(project string, issueDetails map[string]interface{}) (string, error) {
	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": project,
			},
			"summary":     issueDetails["summary"],
			"description": issueDetails["description"],
			"issuetype":   issueDetails["issuetype"],
		},
	}

	resp, err := j.client.R().
		SetBody(payload).
		SetResult(&JiraIssue{}).
		Post("/rest/api/3/issue")

	if err != nil {
		return "", fmt.Errorf("failed to create Jira issue: %v", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return "", fmt.Errorf("Jira issue creation failed with status %d: %s", 
			resp.StatusCode(), resp.String())
	}

	createdIssue := resp.Result().(*JiraIssue)
	return createdIssue.Key, nil
}