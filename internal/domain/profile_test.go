package domain

import "testing"

func TestExperienceFields(t *testing.T) {
	e := Experience{
		Company: "Acme",
		Role:    "Engineer",
		Start:   "2020-01",
		End:     "2022-06",
		Notes:   "some notes",
	}
	if e.Company == "" || e.Role == "" || e.Start == "" {
		t.Error("Experience fields must be assignable")
	}
}

func TestEducationFields(t *testing.T) {
	ed := Education{
		Institution: "MIT",
		Degree:      "BSc",
		Field:       "CS",
		Year:        "2019",
	}
	if ed.Institution == "" || ed.Degree == "" {
		t.Error("Education fields must be assignable")
	}
}

func TestProfileZeroValue(t *testing.T) {
	var p Profile
	_ = p // must compile; zero value is valid
}
