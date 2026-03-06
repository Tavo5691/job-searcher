// Package llm provides adapters for LLM providers implementing port.LLMProvider.
package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ClaudeProvider implements port.LLMProvider using the Anthropic Claude API.
type ClaudeProvider struct {
	client anthropic.Client
}

// NewClaudeProvider creates a new ClaudeProvider with the given API key.
// Returns an error if the key is empty.
func NewClaudeProvider(apiKey string) (*ClaudeProvider, error) {
	if apiKey == "" {
		return nil, errors.New("claude: API key must not be empty")
	}
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &ClaudeProvider{client: client}, nil
}

// Complete sends prompt to Claude and returns the text response.
func (p *ClaudeProvider) Complete(ctx context.Context, prompt string) (string, error) {
	msg, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5HaikuLatest,
		MaxTokens: 4096,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("claude complete: %w", err)
	}

	var sb strings.Builder
	for _, block := range msg.Content {
		if block.Type == "text" {
			sb.WriteString(block.AsText().Text)
		}
	}
	return sb.String(), nil
}
