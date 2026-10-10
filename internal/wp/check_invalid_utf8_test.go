package wp_test

import (
	"testing"
	"unicode/utf8"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestCheckInvalidUTF8Valid(t *testing.T) {
	in := "just a test ✏"
	if got := wp.CheckInvalidUTF8(in, false); got != in {
		t.Fatalf("valid passthrough: got %q", got)
	}
	if got := wp.CheckInvalidUTF8("", false); got != "" {
		t.Fatalf("empty: got %q", got)
	}
}

func TestCheckInvalidUTF8Reject(t *testing.T) {
	invalid := "the byte \xC0 is never allowed in a UTF-8 string."
	if got := wp.CheckInvalidUTF8(invalid, false); got != "" {
		t.Fatalf("expected empty reject, got %q", got)
	}
}

func TestCheckInvalidUTF8Strip(t *testing.T) {
	invalid := "the byte \xC0 is never allowed in a UTF-8 string."
	got := wp.CheckInvalidUTF8(invalid, true)
	want := "the byte " + string(utf8.RuneError) + " is never allowed in a UTF-8 string."
	if got != want {
		t.Fatalf("strip: got %q want %q", got, want)
	}
	if !utf8.ValidString(got) {
		t.Fatal("stripped result must be valid UTF-8")
	}
}
