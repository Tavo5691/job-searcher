package domain

import "testing"

func TestHuntStatusConstants(t *testing.T) {
	tests := []struct {
		name   string
		status HuntStatus
	}{
		{"active", HuntStatusActive},
		{"closed", HuntStatusClosed},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.status == "" {
				t.Errorf("HuntStatus %s must not be empty", tc.name)
			}
		})
	}

	if HuntStatusActive == HuntStatusClosed {
		t.Error("HuntStatusActive and HuntStatusClosed must be distinct")
	}
}

func TestHuntZeroValue(t *testing.T) {
	var h Hunt
	_ = h // must compile; zero value is valid
}
