package domain

import "time"

// Insight represents LLM-generated structured advice for an Application.
// One Insight per Application; overwritten on regeneration.
type Insight struct {
	ID            string
	ApplicationID string
	Content       string // structured markdown advice
	GeneratedAt   time.Time
}
