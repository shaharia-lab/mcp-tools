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
