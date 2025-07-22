package ynab

import (
	"encoding/json"
	"testing"
)

func TestFlagColorJSONMarshaling(t *testing.T) {
	tests := []struct {
		name     string
		color    FlagColor
		expected string
	}{
		{"Red", FlagColorRed, `"red"`},
		{"Orange", FlagColorOrange, `"orange"`},
		{"Yellow", FlagColorYellow, `"yellow"`},
		{"Green", FlagColorGreen, `"green"`},
		{"Blue", FlagColorBlue, `"blue"`},
		{"Purple", FlagColorPurple, `"purple"`},
		{"Empty", FlagColorEmpty, `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test MarshalJSON
			b, err := json.Marshal(tt.color)
			if err != nil {
				t.Fatalf("MarshalJSON failed: %v", err)
			}
			if string(b) != tt.expected {
				t.Errorf("MarshalJSON = %s, want %s", string(b), tt.expected)
			}

			// Test UnmarshalJSON
			var color FlagColor
			err = json.Unmarshal([]byte(tt.expected), &color)
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}
			if color != tt.color {
				t.Errorf("UnmarshalJSON = %v, want %v", color, tt.color)
			}
		})
	}
}

func TestFlagColorUnmarshalNullVariations(t *testing.T) {
	nullVariations := []string{"null", `""`}
	for _, nullStr := range nullVariations {
		t.Run("Unmarshal_"+nullStr, func(t *testing.T) {
			var color FlagColor
			err := json.Unmarshal([]byte(nullStr), &color)
			if err != nil {
				t.Fatalf("UnmarshalJSON failed: %v", err)
			}
			if color != FlagColorEmpty {
				t.Errorf("UnmarshalJSON = %v, want %v", color, FlagColorEmpty)
			}
		})
	}
}
