// Package domain contains the core entities and business rules of the job-search domain.
// It has no external dependencies — only stdlib.
package domain

import "time"

// HuntStatus represents the lifecycle state of a Hunt.
type HuntStatus string

const (
	// HuntStatusActive indicates the hunt is currently in progress.
	HuntStatusActive HuntStatus = "active"
	// HuntStatusClosed indicates the hunt has ended.
	HuntStatusClosed HuntStatus = "closed"
)

// Hunt is a job-searching period. It contains a Profile and all Applications.
type Hunt struct {
	ID        string
	Title     string
	Status    HuntStatus
	CreatedAt time.Time
	ClosedAt  *time.Time
}
