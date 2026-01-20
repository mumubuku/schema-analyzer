package analyzer

import (
	"testing"
)

func TestCalculateNameSimilarity(t *testing.T) {
	r := &RelationshipInferer{}
	
	tests := []struct {
		name1    string
		name2    string
		expected float64
		minScore float64
	}{
		{"cDepCode", "cDepCode", 1.0, 1.0},
		{"cDepCode", "DepCode", 0.8, 0.7},
		{"DepartmentID", "DepID", 0.0, 0.0},
		{"UserID", "UserId", 0.9, 0.8},
	}
	
	for _, tt := range tests {
		t.Run(tt.name1+"_"+tt.name2, func(t *testing.T) {
			score := r.calculateNameSimilarity(tt.name1, tt.name2)
			if tt.expected > 0 {
				if score != tt.expected {
					t.Errorf("expected %f, got %f", tt.expected, score)
				}
			} else {
				if score < tt.minScore {
					t.Errorf("expected >= %f, got %f", tt.minScore, score)
				}
			}
		})
	}
}

func TestIsTypeCompatible(t *testing.T) {
	r := &RelationshipInferer{}
	
	tests := []struct {
		type1    string
		type2    string
		expected bool
	}{
		{"varchar", "varchar", true},
		{"varchar", "nvarchar", true},
		{"int", "bigint", true},
		{"varchar", "int", false},
		{"text", "varchar", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.type1+"_"+tt.type2, func(t *testing.T) {
			result := r.isTypeCompatible(tt.type1, tt.type2)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
