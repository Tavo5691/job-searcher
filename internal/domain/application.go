package domain

import "time"

// ApplicationStatus represents the current state of a job application.
type ApplicationStatus string

const (
	ApplicationStatusApplied      ApplicationStatus = "applied"
	ApplicationStatusInterviewing ApplicationStatus = "interviewing"
	ApplicationStatusOffer        ApplicationStatus = "offer"
	ApplicationStatusAccepted     ApplicationStatus = "accepted"
	ApplicationStatusRejected     ApplicationStatus = "rejected"
	ApplicationStatusWithdrawn    ApplicationStatus = "withdrawn"
)

// Application represents one company/role being pursued within a Hunt.
type Application struct {
	ID             string
	HuntID         string
	CompanyName    string
	RoleTitle      string
	JobDescription string
	Status         ApplicationStatus
	AppliedAt      time.Time
	UpdatedAt      time.Time
	Notes          string
}
