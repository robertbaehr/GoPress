package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P2.162: GetDefaultExtensionForMimeType picks the first extension for a mime.
func TestGetDefaultExtensionForMimeTypeP2162(t *testing.T) {
	if got := wp.GetDefaultExtensionForMimeType(nil, "image/jpeg"); got != "jpg" {
		t.Fatalf("jpeg=%q want jpg", got)
	}
	if got := wp.GetDefaultExtensionForMimeType(nil, "image/png"); got != "png" {
		t.Fatalf("png=%q", got)
	}
	if got := wp.GetDefaultExtensionForMimeType(nil, "application/pdf"); got != "pdf" {
		t.Fatalf("pdf=%q", got)
	}
	if got := wp.GetDefaultExtensionForMimeType(nil, "no/such-mime"); got != "" {
		t.Fatalf("unknown=%q", got)
	}
	if got := wp.GetDefaultExtensionForMimeType(nil, ""); got != "" {
		t.Fatalf("empty=%q", got)
	}
}
