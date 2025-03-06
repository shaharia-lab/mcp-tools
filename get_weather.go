package mcptools

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/shaharia-lab/goai/mcp"
	"github.com/shaharia-lab/goai/observability"
	"go.opentelemetry.io/otel/attribute"
)

// GetWeather is a tool that provides the current weather for a specified location.
// The tool expects an input schema that includes a "location" field, which
// specifies the city and state (e.g., "San Francisco, CA"). It returns the
// weather information as text content.
var GetWeather = mcp.Tool{
	Name:        "get_weather",
	Description: "Get the current weather for a given location.",
	InputSchema: json.RawMessage(`{
				"type": "object",
				"properties": {
					"location": {
						"type": "string",
						"description": "The city and state, e.g. San Francisco, CA"
					}
				},
				"required": ["location"]
			}`),
	Handler: func(ctx context.Context, params mcp.CallToolParams) (mcp.CallToolResult, error) {
		ctx, span := observability.StartSpan(ctx, fmt.Sprintf("%s.Handler", params.Name))
		span.SetAttributes(
			attribute.String("tool_name", params.Name),
			attribute.String("tool_argument", string(params.Arguments)),
		)
		defer span.End()

		var err error
		defer func() {
			if err != nil {
				span.RecordError(err)
			}
		}()

		var input struct {
			Location string `json:"location"`
		}
		if err := json.Unmarshal(params.Arguments, &input); err != nil {
			return mcp.CallToolResult{}, err
		}

		// Return result
		return mcp.CallToolResult{
			Content: []mcp.ToolResultContent{
				{
					Type: "text",
					Text: fmt.Sprintf("Weather in %s: Sunny, 72°F", input.Location),
				},
			},
		}, nil
	},
}
