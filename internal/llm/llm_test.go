// Package llm provides adapters for LLM providers.
package llm

import (
	"testing"

	"github.com/Tavo5691/job-searcher/internal/port"
)

// Compile-time interface checks — no real network calls.
var _ port.LLMProvider = (*ClaudeProvider)(nil)
var _ port.LLMProvider = (*GeminiProvider)(nil)

func TestNewClaudeProviderValidatesKey(t *testing.T) {
	_, err := NewClaudeProvider("")
	if err == nil {
		t.Error("NewClaudeProvider with empty key must return an error")
	}
}

func TestNewGeminiProviderValidatesKey(t *testing.T) {
	_, err := NewGeminiProvider("")
	if err == nil {
		t.Error("NewGeminiProvider with empty key must return an error")
	}
}

func TestNewClaudeProviderSuccess(t *testing.T) {
	p, err := NewClaudeProvider("sk-test-key")
	if err != nil {
		t.Fatalf("NewClaudeProvider with valid key: %v", err)
	}
	if p == nil {
		t.Error("NewClaudeProvider must return non-nil provider")
	}
}

func TestNewGeminiProviderSuccess(t *testing.T) {
	p, err := NewGeminiProvider("AIza-test-key")
	if err != nil {
		t.Fatalf("NewGeminiProvider with valid key: %v", err)
	}
	if p == nil {
		t.Error("NewGeminiProvider must return non-nil provider")
	}
}
