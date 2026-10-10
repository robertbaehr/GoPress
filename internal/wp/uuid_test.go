package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestIsUUID(t *testing.T) {
	cases := []struct {
		uuid    string
		version int
		want    bool
	}{
		{"123e4567-e89b-12d3-a456-426614174000", 0, true},
		{"123E4567-E89B-12D3-A456-426614174000", 0, true},
		{"123e4567-e89b-42d3-a456-426614174000", 4, true},
		{"123e4567-e89b-42d3-8456-426614174000", 4, true},
		{"123e4567-e89b-42d3-c456-426614174000", 4, false}, // variant nibble
		{"123e4567-e89b-12d3-a456-426614174000", 4, false}, // not v4
		{"not-a-uuid", 0, false},
		{"", 0, false},
		{"123e4567e89b12d3a456426614174000", 0, false},
		{"123e4567-e89b-12d3-a456-426614174000", 3, false},
	}
	for _, tc := range cases {
		got := wp.IsUUID(tc.uuid, tc.version)
		if got != tc.want {
			t.Fatalf("IsUUID(%q, %d)=%v want %v", tc.uuid, tc.version, got, tc.want)
		}
	}
}
