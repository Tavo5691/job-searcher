package domain

import "testing"

func TestStageTypeConstants(t *testing.T) {
	types := []StageType{
		StageTypeRecruiterScreen,
		StageTypeTechnicalScreen,
		StageTypeTakeHome,
		StageTypeTechnicalInterview,
		StageTypeBehavioral,
		StageTypeSystemDesign,
		StageTypeOffer,
		StageTypeOther,
	}
	seen := map[StageType]bool{}
	for _, st := range types {
		if st == "" {
			t.Error("StageType constant must not be empty")
		}
		if seen[st] {
			t.Errorf("duplicate StageType: %s", st)
		}
		seen[st] = true
	}
}

func TestStageOutcomeConstants(t *testing.T) {
	outcomes := []StageOutcome{
		StageOutcomePending,
		StageOutcomePassed,
		StageOutcomeFailed,
	}
	seen := map[StageOutcome]bool{}
	for _, o := range outcomes {
		if o == "" {
			t.Error("StageOutcome constant must not be empty")
		}
		if seen[o] {
			t.Errorf("duplicate StageOutcome: %s", o)
		}
		seen[o] = true
	}
}

func TestStageZeroValue(t *testing.T) {
	var s Stage
	_ = s
}
