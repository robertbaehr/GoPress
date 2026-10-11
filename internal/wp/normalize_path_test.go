package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestNormalizePathSeparators(t *testing.T) {
	wp.ResetNormalizePathCacheForTest()
	cases := []struct{ in, want string }{
		{`C:\Windows\Fonts`, `C:/Windows/Fonts`},
		{`c:\Windows\\Fonts`, `C:/Windows/Fonts`},
		{`/var//www///html`, `/var/www/html`},
		{`//server/share//dir`, `//server/share/dir`},
		{`http://example.com//path///x`, `http://example.com/path/x`},
		{`php://filter/read=string.toupper/resource=file`, `php://filter/read=string.toupper/resource=file`},
		{``, ``},
	}
	for _, tc := range cases {
		if got := wp.NormalizePath(tc.in); got != tc.want {
			t.Fatalf("NormalizePath(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}

func TestNormalizePathCached(t *testing.T) {
	wp.ResetNormalizePathCacheForTest()
	a := wp.NormalizePath(`/a//b`)
	b := wp.NormalizePath(`/a//b`)
	if a != `/a/b` || b != a {
		t.Fatalf("cache mismatch %q %q", a, b)
	}
}

func TestIsStream(t *testing.T) {
	wp.ResetStreamWrappersForTest()
	if !wp.IsStream("https://example.com/x") {
		t.Fatal("expected https stream")
	}
	if wp.IsStream("/var/www") {
		t.Fatal("path is not a stream")
	}
	if wp.IsStream("notascheme://x") {
		t.Fatal("unknown wrapper should be false")
	}
	// P2.80: PHP in_array(..., true) is case-sensitive for wrappers.
	if wp.IsStream("HTTPS://example.com/x") {
		t.Fatal("uppercase HTTPS should not match known wrappers")
	}
	if wp.IsStream("File://tmp") {
		t.Fatal("mixed-case File should not match")
	}
}

// P2.116: RegisterStreamWrapper mirrors stream_wrapper_register for wp_is_stream.
func TestRegisterStreamWrapper(t *testing.T) {
	wp.ResetStreamWrappersForTest()
	t.Cleanup(wp.ResetStreamWrappersForTest)
	if wp.IsStream("s3://bucket/key") {
		t.Fatal("s3 should be unregistered initially")
	}
	wp.RegisterStreamWrapper("s3")
	if !wp.IsStream("s3://bucket/key") {
		t.Fatal("expected s3 after register")
	}
	// Case-sensitive: S3 != s3.
	if wp.IsStream("S3://bucket/key") {
		t.Fatal("uppercase S3 should not match")
	}
	wp.UnregisterStreamWrapper("s3")
	if wp.IsStream("s3://bucket/key") {
		t.Fatal("s3 should be gone after unregister")
	}
	// NormalizePath must preserve registered wrappers.
	wp.RegisterStreamWrapper("s3")
	wp.ResetNormalizePathCacheForTest()
	if got := wp.NormalizePath(`s3://bucket//a///b`); got != `s3://bucket/a/b` {
		t.Fatalf("normalize registered stream=%q", got)
	}
}

func TestNormalizePathDriveAndUNC(t *testing.T) {
	wp.ResetNormalizePathCacheForTest()
	if got := wp.NormalizePath(`d:\a\\b`); got != `D:/a/b` {
		t.Fatalf("drive=%q", got)
	}
	if got := wp.NormalizePath(`///unc/share`); got != `//unc/share` {
		t.Fatalf("unc collapse=%q", got)
	}
	// Uppercase scheme is not a stream wrapper, so "://" collapses like any path.
	if got := wp.NormalizePath(`HTTPS://ex.com//a`); got != `HTTPS:/ex.com/a` {
		t.Fatalf("non-stream scheme=%q", got)
	}
}
