package port

import "context"

// LLMProvider is the interface for Large Language Model completions.
// Implementations: internal/llm.ClaudeProvider, internal/llm.GeminiProvider.
type LLMProvider interface {
	Complete(ctx context.Context, prompt string) (string, error)
}
