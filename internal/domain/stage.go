package domain

import "time"

// StageType represents the kind of hiring stage.
type StageType string

const (
	StageTypeRecruiterScreen    StageType = "recruiter_screen"
	StageTypeTechnicalScreen    StageType = "technical_screen"
	StageTypeTakeHome           StageType = "take_home"
	StageTypeTechnicalInterview StageType = "technical_interview"
	StageTypeBehavioral         StageType = "behavioral"
	StageTypeSystemDesign       StageType = "system_design"
	StageTypeOffer              StageType = "offer"
	StageTypeOther              StageType = "other"
)

// StageOutcome represents the result of a hiring stage.
type StageOutcome string

const (
	StageOutcomePending StageOutcome = "pending"
	StageOutcomePassed  StageOutcome = "passed"
	StageOutcomeFailed  StageOutcome = "failed"
)

// Stage represents a single step in an Application's hiring process.
type Stage struct {
	ID            string
	ApplicationID string
	Type          StageType
	Label         string // used when Type is StageTypeOther
	Date          *time.Time
	Notes         string
	Feedback      string
	Outcome       StageOutcome
	Order         int
}
