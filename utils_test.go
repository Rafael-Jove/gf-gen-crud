package main

import "testing"

func TestShortName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Users", "User"},
		{"Statuses", "Status"},
		{"Status", "Status"},
		{"Campuses", "Campus"},
		{"Campus", "Campus"},
		{"Processes", "Process"},
		{"Process", "Process"},
		{"Addresses", "Address"},
		{"Address", "Address"},
		{"Businesses", "Business"},
		{"News", "News"},
		{"Series", "Series"},
		{"Species", "Species"},
		{"Buses", "Bus"},
		{"Bus", "Bus"},
		{"Categories", "Category"},
		{"Companies", "Company"},
		{"Boxes", "Box"},
		{"Quizzes", "Quiz"},
		{"Analyses", "Analysis"},
		{"PersonalProfiles", "PersonalProfile"},
		{"PersonalProfile", "PersonalProfile"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shortName(tt.input)
			if got != tt.expected {
				t.Errorf("shortName(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
