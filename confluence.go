package mcptools

import (
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
)

// ConfluenceClient represents a client for interacting with Confluence
type ConfluenceClient struct {
	BaseURL   string
	AuthToken string
	client    *resty.Client
}

// ConfluencePage represents the structure of a Confluence page
type ConfluencePage struct {
	ID     string            `json:"id"`
	Type   string            `json:"type"`
	Title  string            `json:"title"`
	Status string            `json:"status"`
	Body   map[string]interface{} `json:"body,omitempty"`
	Version map[string]interface{} `json:"version,omitempty"`
}

// NewConfluenceClient creates a new Confluence client
func NewConfluenceClient(baseURL, authToken string) *ConfluenceClient {
	client := resty.New()
	client.SetBaseURL(baseURL)
	client.SetHeader("Authorization", fmt.Sprintf("Bearer %s", authToken))
	client.SetHeader("Content-Type", "application/json")

	return &ConfluenceClient{
		BaseURL:   baseURL,
		AuthToken: authToken,
		client:    client,
	}
}

// SearchConfluencePages searches for Confluence pages based on a query
func (c *ConfluenceClient) SearchConfluencePages(query string) ([]ConfluencePage, error) {
	resp, err := c.client.R().
		SetQueryParams(map[string]string{
			"cql":   query,
			"limit": "100", // Set a default limit
		}).
		SetResult(&struct {
			Results []ConfluencePage `json:"results"`
		}{}).
		Get("/wiki/rest/api/search")

	if err != nil {
		return nil, fmt.Errorf("failed to search Confluence pages: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Confluence search request failed with status %d: %s", 
			resp.StatusCode(), resp.String())
	}

	result := resp.Result().(*struct {
		Results []ConfluencePage `json:"results"`
	})

	return result.Results, nil
}

// ReadConfluencePage retrieves details of a specific Confluence page
func (c *ConfluenceClient) ReadConfluencePage(pageID string) (*ConfluencePage, error) {
	resp, err := c.client.R().
		SetPathParams(map[string]string{
			"pageId": pageID,
		}).
		SetResult(&ConfluencePage{}).
		SetQueryParams(map[string]string{
			"expand": "body.storage,version", // Optional: expand body and version details
		}).
		Get("/wiki/rest/api/content/{pageId}")

	if err != nil {
		return nil, fmt.Errorf("failed to read Confluence page: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("Confluence page read failed with status %d: %s", 
			resp.StatusCode(), resp.String())
	}

	page := resp.Result().(*ConfluencePage)
	return page, nil
}

// CreateConfluencePage creates a new page in Confluence
func (c *ConfluenceClient) CreateConfluencePage(spaceKey string, title string, content string) (*ConfluencePage, error) {
	payload := map[string]interface{}{
		"type": "page",
		"title": title,
		"space": map[string]string{
			"key": spaceKey,
		},
		"body": map[string]interface{}{
			"storage": map[string]string{
				"value":          content,
				"representation": "storage",
			},
		},
	}

	resp, err := c.client.R().
		SetBody(payload).
		SetResult(&ConfluencePage{}).
		Post("/wiki/rest/api/content")

	if err != nil {
		return nil, fmt.Errorf("failed to create Confluence page: %v", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("Confluence page creation failed with status %d: %s", 
			resp.StatusCode(), resp.String())
	}

	createdPage := resp.Result().(*ConfluencePage)
	return createdPage, nil
}