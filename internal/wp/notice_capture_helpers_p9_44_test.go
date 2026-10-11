package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P9.44: BeginNoticeCapture / WriteNotice / EndNoticeCapture Context helpers.
func TestNoticeCaptureHelpersP944(t *testing.T) {
	c := testBootQuery(t)
	var buf strings.Builder
	wp.BeginNoticeCapture(c, &buf)
	wp.WriteNotice(c, "<p>hello</p>")
	wp.EndNoticeCapture(c)
	if got := buf.String(); !strings.Contains(got, "<p>hello</p>") {
		t.Fatalf("buf=%q", got)
	}
	// After end, further writes are ignored.
	wp.WriteNotice(c, "<p>ignored</p>")
	if strings.Contains(buf.String(), "ignored") {
		t.Fatalf("wrote after end: %q", buf.String())
	}
	wp.BeginNoticeCapture(nil, &buf)
	wp.WriteNotice(nil, "x")
	wp.EndNoticeCapture(nil)
}
