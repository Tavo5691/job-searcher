// main.go wires all adapters, validates configuration, and starts the TUI.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/Tavo5691/job-searcher/internal/app"
	"github.com/Tavo5691/job-searcher/internal/llm"
	"github.com/Tavo5691/job-searcher/internal/pdf"
	"github.com/Tavo5691/job-searcher/internal/port"
	"github.com/Tavo5691/job-searcher/internal/store"
	"github.com/Tavo5691/job-searcher/internal/tui"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	// ── Config ──────────────────────────────────────────────────────────────
	dbPath := envOr("DB_PATH", "job-searcher.db")
	llmProvider := envOr("LLM_PROVIDER", "")

	if llmProvider == "" {
		return errors.New("LLM_PROVIDER env var is required (claude or gemini)")
	}

	// ── Store adapter ───────────────────────────────────────────────────────
	s, err := store.New(dbPath)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer s.Close()

	// ── LLM adapter ─────────────────────────────────────────────────────────
	var provider port.LLMProvider
	switch llmProvider {
	case "claude":
		apiKey := os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return errors.New("ANTHROPIC_API_KEY is required when LLM_PROVIDER=claude")
		}
		provider, err = llm.NewClaudeProvider(apiKey)
		if err != nil {
			return fmt.Errorf("init claude provider: %w", err)
		}
	case "gemini":
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			return errors.New("GEMINI_API_KEY is required when LLM_PROVIDER=gemini")
		}
		provider, err = llm.NewGeminiProvider(apiKey)
		if err != nil {
			return fmt.Errorf("init gemini provider: %w", err)
		}
	default:
		return fmt.Errorf("unknown LLM_PROVIDER %q — must be claude or gemini", llmProvider)
	}

	// ── PDF adapter ──────────────────────────────────────────────────────────
	pdfParser := pdf.NewPdfcpuParser()

	// ── Application service ──────────────────────────────────────────────────
	svc := app.NewService(s, provider, pdfParser)

	// ── TUI ──────────────────────────────────────────────────────────────────
	a := tui.NewApp(svc)
	return a.Run()
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
