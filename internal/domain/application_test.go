package domain

import "testing"

func TestApplicationStatusConstants(t *testing.T) {
	statuses := []ApplicationStatus{
		ApplicationStatusApplied,
		ApplicationStatusInterviewing,
		ApplicationStatusOffer,
		ApplicationStatusAccepted,
		ApplicationStatusRejected,
		ApplicationStatusWithdrawn,
	}
	seen := map[ApplicationStatus]bool{}
	for _, s := range statuses {
		if s == "" {
			t.Errorf("ApplicationStatus constant must not be empty")
		}
		if seen[s] {
			t.Errorf("duplicate ApplicationStatus value: %s", s)
		}
		seen[s] = true
	}
}

func TestApplicationZeroValue(t *testing.T) {
	var a Application
	_ = a
}
