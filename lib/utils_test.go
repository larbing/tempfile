package lib

import (
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"Length8", 8},
		{"Length10", 10},
		{"Length20", 20},
		{"Length64", 64},
		{"LengthExceedsHash", 100}, // Exceeds the hash length
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := GenerateID(tt.length)

			// Check if the length of the generated ID matches the expected length
			expectedLength := tt.length
			if tt.length > 64 { // SHA-256 hash length in hex is 64
				expectedLength = 64
			}
			if len(id) != expectedLength {
				t.Errorf("expected length %d, got %d", expectedLength, len(id))
			}

			// Check if the ID contains only valid hexadecimal characters
			for _, char := range id {
				if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
					t.Errorf("invalid character '%c' in ID", char)
				}
			}
		})
	}
}

func TestGenerateIDUniqueness(t *testing.T) {
	id1 := GenerateID(8)
	id2 := GenerateID(8)

	if id1 == id2 {
		t.Errorf("expected unique IDs, but got identical IDs: %s and %s", id1, id2)
	}
}
