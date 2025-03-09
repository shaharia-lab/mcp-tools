package mcptools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"google.golang.org/api/gmail/v1"
	"time"

	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"go.opentelemetry.io/otel/attribute"
)

const (
	GmailToolName = "gmail_all_in_one"
	// Gmail API scopes
	GmailReadScope   = gmail.GmailReadonlyScope // For reading emails
	GmailSendScope   = gmail.GmailSendScope     // For sending emails
	GmailModifyScope = gmail.GmailModifyScope   // For modifying/deleting emails
	GmailFullScope   = gmail.MailGoogleComScope // Full access
)

// Gmail represents a wrapper around the Gmail API service,
// providing a programmatic interface for executing Gmail operations.
type Gmail struct {
	logger  observability.Logger
	service *gmail.Service
	config  GmailConfig
}

type EmailMessage struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	Subject string `json:"subject"`
	Snippet string `json:"snippet"`
	Date    string `json:"date"`
}

// GmailConfig holds the configuration for the Gmail tool
type GmailConfig struct {
	UserID         string
	MaxResults     int64
	SinceLastNDays int
}

// NewGmail creates and returns a new instance of the Gmail wrapper with the provided configuration.
func NewGmail(logger observability.Logger, service *gmail.Service, config GmailConfig) *Gmail {
	return &Gmail{
		logger:  logger,
		service: service,
		config:  config,
	}
}

// GmailAllInOneTool returns a mcp.Tool that can perform various Gmail operations
func (g *Gmail) GmailAllInOneTool() mcp.Tool {
	return mcp.Tool{
		Name:        GmailToolName,
		Description: "Performs Gmail operations such as list, send, read messages",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"operation": {
					"type": "string",
					"description": "Gmail operation to execute (list, send, read, delete)",
					"enum": ["list", "send", "read"]
				},
				"message_id": {
					"type": "string",
					"description": "Message ID for read/delete operations"
				},
				"query": {
					"type": "string",
					"description": "Search query for list operation"
				},
				"email": {
					"type": "object",
					"properties": {
						"to": {
							"type": "string",
							"description": "Recipient email address"
						},
						"subject": {
							"type": "string",
							"description": "Email subject"
						},
						"body": {
							"type": "string",
							"description": "Email body content"
						}
					}
				},
				"max_results": {
					"type": "integer",
					"description": "Maximum number of results to return"
				},
				"days": {
					"type": "integer",
					"description": "Consider messages since the last N days. Maximum 20 days allowed"
				}
			},
			"required": ["operation"]
		}`),
		Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
			ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
			span.SetAttributes(
				attribute.String("tool_name", params.Name),
				attribute.String("tool_argument", string(params.Arguments)),
			)
			defer span.End()

			g.logger.WithFields(map[string]interface{}{
				"tool_name": params.Name,
				"arguments": string(params.Arguments),
			}).Info("Starting Gmail operation execution")

			var input struct {
				Operation  string `json:"operation"`
				MessageID  string `json:"message_id,omitempty"`
				Query      string `json:"query,omitempty"`
				Days       int    `json:"days,omitempty"`
				MaxResults int64  `json:"max_results,omitempty"`
				Email      struct {
					To      string `json:"to,omitempty"`
					Subject string `json:"subject,omitempty"`
					Body    string `json:"body,omitempty"`
				} `json:"email,omitempty"`
			}

			if err := json.Unmarshal(params.Arguments, &input); err != nil {
				g.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"raw_input":                 string(params.Arguments),
				}).Error("Failed to unmarshal input parameters")

				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("failed to unmarshal input: %w", err)
			}

			var result string
			var err error

			switch input.Operation {
			case "list":
				result, err = g.listMessages(ctx, input.Query, input.Days, input.MaxResults)
			case "send":
				result, err = g.sendMessage(ctx, input.Email.To, input.Email.Subject, input.Email.Body)
			case "read":
				result, err = g.readMessage(ctx, input.MessageID)
			default:
				err = fmt.Errorf("unsupported operation: %s", input.Operation)
			}

			if err != nil {
				g.logger.WithFields(map[string]interface{}{
					observability.ErrorLogField: err,
					"operation":                 input.Operation,
				}).Error("Gmail operation failed")

				span.RecordError(err)
				return mcp.CallToolResult{}, fmt.Errorf("gmail %s error: %w", input.Operation, err)
			}

			g.logger.WithFields(map[string]interface{}{
				"operation": input.Operation,
				"result":    result,
			}).Debug("Gmail operation completed successfully")

			return mcp.CallToolResult{
				Content: []mcp.ToolResultContent{{
					Type: "text",
					Text: result,
				}},
			}, nil
		},
	}
}

func (g *Gmail) listMessages(ctx context.Context, query string, days int, maxResults int64) (string, error) {
	// If days parameter is provided, add date range to query
	if days > 0 {
		// Calculate the date from X days ago
		fromDate := time.Now().AddDate(0, 0, -days)
		dateQuery := fmt.Sprintf("after:%s", fromDate.Format("2006/01/02"))

		if query != "" {
			query = fmt.Sprintf("%s %s", dateQuery, query)
		} else {
			query = dateQuery
		}
	}

	// Create the list request
	req := g.service.Users.Messages.List("me")
	if query != "" {
		req = req.Q(query)
	}

	req = req.MaxResults(20)
	if maxResults > 0 {
		req = req.MaxResults(maxResults)
	}

	resp, err := req.Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to list messages: %w", err)
	}

	var messages []EmailMessage
	for _, msg := range resp.Messages {
		// Get full message details
		fullMsg, err := g.service.Users.Messages.Get("me", msg.Id).
			Format("full").
			Context(ctx).
			Do()
		if err != nil {
			g.logger.WithFields(map[string]interface{}{
				observability.ErrorLogField: err,
				"message_id":                msg.Id,
			}).Error("Failed to fetch message details")
			continue
		}

		// Extract headers
		var from, subject, date string
		if fullMsg.Payload != nil {
			for _, header := range fullMsg.Payload.Headers {
				switch header.Name {
				case "From":
					from = header.Value
				case "Subject":
					subject = header.Value
				case "Date":
					date = header.Value
				}
			}
		}

		messages = append(messages, EmailMessage{
			ID:      fullMsg.Id,
			From:    from,
			Subject: subject,
			Snippet: fullMsg.Snippet,
			Date:    date,
		})
	}

	// If no messages found
	if len(messages) == 0 {
		return "No messages found", nil
	}

	// Convert to JSON for formatted output
	jsonOutput, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format messages: %w", err)
	}

	return string(jsonOutput), nil
}

func (g *Gmail) sendMessage(ctx context.Context, to, subject, body string) (string, error) {
	// Implement email sending logic using gmail.Service
	// This is a simplified version - you'll need to properly construct the email
	message := gmail.Message{
		Raw: createEncodedEmail(to, subject, body), // You'll need to implement this helper
	}

	resp, err := g.service.Users.Messages.Send("me", &message).Do()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Message sent successfully. ID: %s", resp.Id), nil
}

func (g *Gmail) readMessage(ctx context.Context, messageID string) (string, error) {
	msg, err := g.service.Users.Messages.Get("me", messageID).Do()
	if err != nil {
		return "", err
	}

	// Format the message content as needed
	return fmt.Sprintf("Message snippet: %s", msg.Snippet), nil
}

func createEncodedEmail(to, subject, body string) string {
	// Create email message according to RFC 5322
	message := fmt.Sprintf("From: me\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n"+
		"\r\n"+
		"%s", to, subject, body)

	// Encode to base64URL
	return base64.URLEncoding.EncodeToString([]byte(message))
}
