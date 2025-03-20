package mcptools

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTransport is an http.RoundTripper that returns predefined responses
type MockTransport struct {
	mock.Mock
}

func (m *MockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	return args.Get(0).(*http.Response), args.Error(1)
}

// Helper function to create mock HTTP responses
func createMockResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestCreateIssueTool(t *testing.T) {
	t.Run("Successfully_creates_an_issue", func(t *testing.T) {
		// Create the mock logger but don't set expectations yet
		mockLogger := new(MockLogger)

		// Create a mock transport for HTTP requests
		mockTransport := new(MockTransport)

		// Create the Jira client with mocked dependencies
		client := &JiraClient{
			logger:     mockLogger,
			baseURL:    "https://jira-test.example.com",
			username:   "testuser",
			apiToken:   "testtoken",
			httpClient: &http.Client{Transport: mockTransport},
		}

		// Set up the mock logger to handle the WithFields call that's causing issues
		mockReturnLogger := new(MockLogger)
		mockLogger.On("WithFields", mock.MatchedBy(func(fields map[string]interface{}) bool {
			// Match any call to WithFields with a map containing these specific keys and values
			method, hasMethod := fields["method"]
			tool, hasTool := fields["tool"]
			url, hasUrl := fields["url"]

			return hasMethod && hasTool && hasUrl &&
				method == "POST" &&
				tool == "jira" &&
				url == "https://jira-test.example.com/rest/api/2/issue"
		})).Return(mockReturnLogger)

		// Set up expectations for the returned logger
		// These are likely the methods that will be called after WithFields
		mockReturnLogger.On("Info", mock.Anything).Return()
		mockReturnLogger.On("Error", mock.Anything).Return()
		mockReturnLogger.On("Debug", mock.Anything).Return()

		// Set up successful HTTP response
		successResponse := `{"id":"12345","key":"TEST-123","self":"https://jira-test.example.com/rest/api/2/issue/12345"}`
		mockTransport.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodPost &&
				req.URL.Path == "/rest/api/2/issue" &&
				req.Header.Get("Content-Type") == "application/json"
		})).Return(createMockResponse(201, successResponse), nil)

		// Execute the test
		tool := client.CreateIssueTool()
		params := mcp.CallToolParams{
			Arguments: []byte(`{
				"project": "TEST",
				"issueType": "Bug",
				"summary": "Test issue",
				"description": "This is a test issue",
				"priority": "High",
				"assignee": "testuser",
				"labels": ["test", "unit-test"]
			}`),
		}

		result, err := tool.Handler(context.Background(), params)

		// Verify the result
		assert.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		assert.Equal(t, "text", result.Content[0].Type)
		assert.Contains(t, result.Content[0].Text, "Issue created successfully")
		assert.Contains(t, result.Content[0].Text, "TEST-123")

		// Assert that expected HTTP calls were made
		mockTransport.AssertExpectations(t)

		// Assert only the WithFields method - don't check the returned logger expectations
		// as we can't be sure exactly how the returned logger is being used
		mockLogger.AssertCalled(t, "WithFields", mock.MatchedBy(func(fields map[string]interface{}) bool {
			return fields["method"] == "POST" &&
				fields["tool"] == "jira" &&
				fields["url"] == "https://jira-test.example.com/rest/api/2/issue"
		}))
	})
}

func TestGetIssueTool(t *testing.T) {
	t.Run("Successfully_gets_an_issue", func(t *testing.T) {
		// Create the mock logger
		mockLogger := new(MockLogger)

		// Create a mock transport for HTTP requests
		mockTransport := new(MockTransport)

		// Create the Jira client with mocked dependencies
		client := &JiraClient{
			logger:     mockLogger,
			baseURL:    "https://jira-test.example.com",
			username:   "testuser",
			apiToken:   "testtoken",
			httpClient: &http.Client{Transport: mockTransport},
		}

		// Set up the mock logger
		mockReturnLogger := new(MockLogger)
		mockLogger.On("WithFields", mock.MatchedBy(func(fields map[string]interface{}) bool {
			return fields["method"] == "GET" &&
				fields["tool"] == "jira" &&
				fields["url"] == "https://jira-test.example.com/rest/api/2/issue/TEST-123"
		})).Return(mockReturnLogger)

		mockReturnLogger.On("Info", mock.Anything).Return()
		mockReturnLogger.On("Error", mock.Anything).Return()
		mockReturnLogger.On("Debug", mock.Anything).Return()

		// Set up successful HTTP response
		issueJson := `{
			"id": "12345",
			"key": "TEST-123",
			"fields": {
				"summary": "Test issue",
				"description": "This is a test issue",
				"status": {"name": "Open"},
				"priority": {"name": "High"},
				"assignee": {"displayName": "Test User"},
				"reporter": {"displayName": "Reporter User"},
				"created": "2023-01-01T12:00:00.000+0000",
				"updated": "2023-01-02T12:00:00.000+0000"
			}
		}`
		mockTransport.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodGet &&
				strings.Contains(req.URL.String(), "/rest/api/2/issue/TEST-123")
		})).Return(createMockResponse(200, issueJson), nil)

		// Execute the test
		tool := client.GetIssueTool()
		params := mcp.CallToolParams{
			Arguments: []byte(`{"issueKey": "TEST-123"}`),
		}

		result, err := tool.Handler(context.Background(), params)

		// Verify the result
		assert.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Len(t, result.Content, 1)
		assert.Equal(t, "text", result.Content[0].Type)
		assert.Contains(t, result.Content[0].Text, "TEST-123")

		// Assert that expected HTTP calls were made
		mockTransport.AssertExpectations(t)
		mockLogger.AssertCalled(t, "WithFields", mock.Anything)
	})

	t.Run("Issue_not_found", func(t *testing.T) {
		// Create the mock logger
		mockLogger := new(MockLogger)

		// Create a mock transport for HTTP requests
		mockTransport := new(MockTransport)

		// Create the Jira client with mocked dependencies
		client := &JiraClient{
			logger:     mockLogger,
			baseURL:    "https://jira-test.example.com",
			username:   "testuser",
			apiToken:   "testtoken",
			httpClient: &http.Client{Transport: mockTransport},
		}

		// Set up the mock logger
		mockReturnLogger := new(MockLogger)
		mockLogger.On("WithFields", mock.Anything).Return(mockReturnLogger)
		mockReturnLogger.On("Info", mock.Anything).Return()
		mockReturnLogger.On("Error", mock.Anything).Return()
		mockReturnLogger.On("Debug", mock.Anything).Return()

		// Set up HTTP 404 response
		mockTransport.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodGet &&
				strings.Contains(req.URL.String(), "/rest/api/2/issue/INVALID-123")
		})).Return(createMockResponse(404, `{"errorMessages":["Issue does not exist or you do not have permission to see it."]}`), nil)

		// Execute the test
		tool := client.GetIssueTool()
		params := mcp.CallToolParams{
			Arguments: []byte(`{"issueKey": "INVALID-123"}`),
		}

		result, err := tool.Handler(context.Background(), params)

		// Verify the result
		assert.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "Issue does not exist")

		mockTransport.AssertExpectations(t)
	})
}

func TestSearchIssuesTool(t *testing.T) {
	t.Run("Successfully_searches_issues", func(t *testing.T) {
		// Create the mock logger
		mockLogger := new(MockLogger)

		// Create a mock transport for HTTP requests
		mockTransport := new(MockTransport)

		// Create the Jira client with mocked dependencies
		client := &JiraClient{
			logger:     mockLogger,
			baseURL:    "https://jira-test.example.com",
			username:   "testuser",
			apiToken:   "testtoken",
			httpClient: &http.Client{Transport: mockTransport},
		}

		// Set up the mock logger
		mockReturnLogger := new(MockLogger)
		mockLogger.On("WithFields", mock.Anything).Return(mockReturnLogger)
		mockReturnLogger.On("Info", mock.Anything).Return()
		mockReturnLogger.On("Error", mock.Anything).Return()
		mockReturnLogger.On("Debug", mock.Anything).Return()

		// Set up successful HTTP response
		searchJson := `{
			"total": 2,
			"issues": [
				{
					"id": "12345",
					"key": "TEST-123",
					"fields": {
						"summary": "First test issue",
						"status": {"name": "Open"}
					}
				},
				{
					"id": "12346",
					"key": "TEST-124",
					"fields": {
						"summary": "Second test issue",
						"status": {"name": "In Progress"}
					}
				}
			]
		}`
		mockTransport.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodPost &&
				strings.Contains(req.URL.String(), "/rest/api/2/search")
		})).Return(createMockResponse(200, searchJson), nil)

		// Execute the test
		tool := client.SearchIssuesTool()
		params := mcp.CallToolParams{
			Arguments: []byte(`{"jql": "project=TEST AND status=Open", "maxResults": 10}`),
		}

		result, err := tool.Handler(context.Background(), params)

		// Verify the result - updated to check for JSON content rather than specific text
		assert.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Equal(t, "text", result.Content[0].Type)
		// Check that the result contains the JSON response
		assert.Contains(t, result.Content[0].Text, `"total": 2`)
		assert.Contains(t, result.Content[0].Text, `"key": "TEST-123"`)
		assert.Contains(t, result.Content[0].Text, `"key": "TEST-124"`)

		mockTransport.AssertExpectations(t)
	})

	t.Run("No_issues_found", func(t *testing.T) {
		// Create the mock logger
		mockLogger := new(MockLogger)

		// Create a mock transport for HTTP requests
		mockTransport := new(MockTransport)

		// Create the Jira client with mocked dependencies
		client := &JiraClient{
			logger:     mockLogger,
			baseURL:    "https://jira-test.example.com",
			username:   "testuser",
			apiToken:   "testtoken",
			httpClient: &http.Client{Transport: mockTransport},
		}

		// Set up the mock logger
		mockReturnLogger := new(MockLogger)
		mockLogger.On("WithFields", mock.Anything).Return(mockReturnLogger)
		mockReturnLogger.On("Info", mock.Anything).Return()
		mockReturnLogger.On("Debug", mock.Anything).Return()

		// Set up successful HTTP response with no issues
		searchJson := `{"total": 0, "issues": []}`
		mockTransport.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodPost &&
				strings.Contains(req.URL.String(), "/rest/api/2/search")
		})).Return(createMockResponse(200, searchJson), nil)

		// Execute the test
		tool := client.SearchIssuesTool()
		params := mcp.CallToolParams{
			Arguments: []byte(`{"jql": "project=TEST AND status=Resolved", "maxResults": 10}`),
		}

		result, err := tool.Handler(context.Background(), params)

		// Verify the result - updated to check for JSON content rather than specific text
		assert.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, `"total": 0`)
		assert.Contains(t, result.Content[0].Text, `"issues": []`)

		mockTransport.AssertExpectations(t)
	})
}

func TestUpdateIssueTool(t *testing.T) {
	t.Run("Successfully_updates_an_issue", func(t *testing.T) {
		// Create the mock logger
		mockLogger := new(MockLogger)

		// Create a mock transport for HTTP requests
		mockTransport := new(MockTransport)

		// Create the Jira client with mocked dependencies
		client := &JiraClient{
			logger:     mockLogger,
			baseURL:    "https://jira-test.example.com",
			username:   "testuser",
			apiToken:   "testtoken",
			httpClient: &http.Client{Transport: mockTransport},
		}

		// Set up the mock logger
		mockReturnLogger := new(MockLogger)
		mockLogger.On("WithFields", mock.Anything).Return(mockReturnLogger)
		mockReturnLogger.On("Info", mock.Anything).Return()
		mockReturnLogger.On("Debug", mock.Anything).Return()

		// Set up successful HTTP response (PUT typically returns no content)
		mockTransport.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodPut &&
				strings.Contains(req.URL.String(), "/rest/api/2/issue/TEST-123")
		})).Return(createMockResponse(204, ""), nil)

		// Execute the test
		tool := client.UpdateIssueTool()
		params := mcp.CallToolParams{
			Arguments: []byte(`{
				"issueKey": "TEST-123",
				"summary": "Updated summary",
				"description": "Updated description",
				"priority": "Medium"
			}`),
		}

		result, err := tool.Handler(context.Background(), params)

		// Verify the result
		assert.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "Issue TEST-123 updated successfully")

		mockTransport.AssertExpectations(t)
	})
}

func TestAddCommentTool(t *testing.T) {
	t.Run("Successfully_adds_a_comment", func(t *testing.T) {
		// Create the mock logger
		mockLogger := new(MockLogger)

		// Create a mock transport for HTTP requests
		mockTransport := new(MockTransport)

		// Create the Jira client with mocked dependencies
		client := &JiraClient{
			logger:     mockLogger,
			baseURL:    "https://jira-test.example.com",
			username:   "testuser",
			apiToken:   "testtoken",
			httpClient: &http.Client{Transport: mockTransport},
		}

		// Set up the mock logger
		mockReturnLogger := new(MockLogger)
		mockLogger.On("WithFields", mock.Anything).Return(mockReturnLogger)
		mockReturnLogger.On("Info", mock.Anything).Return()
		mockReturnLogger.On("Debug", mock.Anything).Return()

		// Set up successful HTTP response
		commentJson := `{
			"id": "54321",
			"body": "This is a test comment",
			"author": {
				"displayName": "Test User"
			},
			"created": "2023-01-01T12:00:00.000+0000"
		}`
		mockTransport.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodPost &&
				strings.Contains(req.URL.String(), "/rest/api/2/issue/TEST-123/comment")
		})).Return(createMockResponse(201, commentJson), nil)

		// Execute the test
		tool := client.AddCommentTool()
		params := mcp.CallToolParams{
			Arguments: []byte(`{
				"issueKey": "TEST-123",
				"comment": "This is a test comment"
			}`),
		}

		result, err := tool.Handler(context.Background(), params)

		// Verify the result with the actual output format
		assert.NoError(t, err)
		assert.False(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "Comment added successfully")
		assert.Contains(t, result.Content[0].Text, `"id": "54321"`)
		assert.Contains(t, result.Content[0].Text, `"body": "This is a test comment"`)

		mockTransport.AssertExpectations(t)
	})

	t.Run("Failed_to_add_comment", func(t *testing.T) {
		// Create the mock logger
		mockLogger := new(MockLogger)

		// Create a mock transport for HTTP requests
		mockTransport := new(MockTransport)

		// Create the Jira client with mocked dependencies
		client := &JiraClient{
			logger:     mockLogger,
			baseURL:    "https://jira-test.example.com",
			username:   "testuser",
			apiToken:   "testtoken",
			httpClient: &http.Client{Transport: mockTransport},
		}

		// Set up the mock logger
		mockReturnLogger := new(MockLogger)
		mockLogger.On("WithFields", mock.Anything).Return(mockReturnLogger)
		mockReturnLogger.On("Info", mock.Anything).Return()
		mockReturnLogger.On("Error", mock.Anything).Return()
		mockReturnLogger.On("Debug", mock.Anything).Return()

		// Set up error HTTP response
		errorJson := `{"errorMessages":["Issue does not exist or you do not have permission to see it."]}`
		mockTransport.On("RoundTrip", mock.MatchedBy(func(req *http.Request) bool {
			return req.Method == http.MethodPost &&
				strings.Contains(req.URL.String(), "/rest/api/2/issue/INVALID-123/comment")
		})).Return(createMockResponse(404, errorJson), nil)

		// Execute the test
		tool := client.AddCommentTool()
		params := mcp.CallToolParams{
			Arguments: []byte(`{
                "issueKey": "INVALID-123",
                "comment": "This is a test comment"
            }`),
		}

		result, err := tool.Handler(context.Background(), params)

		// Updated assertions to match the actual error message format
		assert.NoError(t, err)
		assert.True(t, result.IsError)
		assert.Contains(t, result.Content[0].Text, "API error (status 404)")
		assert.Contains(t, result.Content[0].Text, "Issue does not exist")

		mockTransport.AssertExpectations(t)
	})
}
