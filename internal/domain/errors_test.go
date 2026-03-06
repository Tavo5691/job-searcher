package domain

import (
	"errors"
	"fmt"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"ErrNotFound", ErrNotFound},
		{"ErrConflict", ErrConflict},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Errorf("%s must not be nil", tc.name)
			}
			// Sentinel errors must be detectable with errors.Is
			wrapped := fmt.Errorf("wrap: %w", tc.err)
			if !errors.Is(wrapped, tc.err) {
				t.Errorf("errors.Is must return true for wrapped %s", tc.name)
			}
		})
	}
}
