package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiProvider implements port.LLMProvider using the Google Gemini API.
type GeminiProvider struct {
	client    *genai.Client
	modelName string
}

// NewGeminiProvider creates a new GeminiProvider with the given API key.
// Returns an error if the key is empty.
func NewGeminiProvider(apiKey string) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, errors.New("gemini: API key must not be empty")
	}
	// Client creation is deferred to actual usage to avoid network calls in constructors.
	// We store the key and create the client lazily; for the scaffold we create it eagerly
	// but do not make any network calls.
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("gemini: create client: %w", err)
	}
	return &GeminiProvider{client: client, modelName: "gemini-1.5-flash"}, nil
}

// Complete sends prompt to Gemini and returns the text response.
func (p *GeminiProvider) Complete(ctx context.Context, prompt string) (string, error) {
	model := p.client.GenerativeModel(p.modelName)
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("gemini complete: %w", err)
	}

	var sb strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content == nil {
			continue
		}
		for _, part := range cand.Content.Parts {
			if t, ok := part.(genai.Text); ok {
				sb.WriteString(string(t))
			}
		}
	}
	return sb.String(), nil
}
