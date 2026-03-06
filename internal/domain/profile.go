package domain

import "time"

// Experience represents a single job or role in a candidate's work history.
type Experience struct {
	Company string
	Role    string
	Start   string
	End     string
	Notes   string
}

// Education represents a single academic credential.
type Education struct {
	Institution string
	Degree      string
	Field       string
	Year        string
}

// Profile holds the experience, education, and skills data for a Hunt.
type Profile struct {
	ID            string
	HuntID        string
	Name          string
	Summary       string
	Skills        []string
	Experience    []Experience
	Education     []Education
	RawResumeText string
	UpdatedAt     time.Time
}
