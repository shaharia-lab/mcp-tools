package mcptools

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/shaharia-lab/goai"
)

func TestGetWeather(t *testing.T) {
	// Define the input parameters
	input := map[string]interface{}{
		"location": "San Francisco, CA",
	}
	inputBytes, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Failed to marshal input: %v", err)
	}

	// Create the tool parameters
	params := goai.CallToolParams{
		Name:      "get_weather",
		Arguments: inputBytes,
	}

	// Call the handler
	result, err := GetWeather.Handler(context.Background(), params)
	if err != nil {
		t.Fatalf("Handler returned an error: %v", err)
	}

	// Check the result
	expectedText := "Weather in San Francisco, CA: Sunny, 72°F"
	if len(result.Content) != 1 || result.Content[0].Text != expectedText {
		t.Errorf("Unexpected result: got %v, want %v", result.Content, expectedText)
	}
}
