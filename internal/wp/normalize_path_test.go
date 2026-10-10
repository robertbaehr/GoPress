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
	if !wp.IsStream("https://example.com/x") {
		t.Fatal("expected https stream")
	}
	if wp.IsStream("/var/www") {
		t.Fatal("path is not a stream")
	}
	if wp.IsStream("notascheme://x") {
		t.Fatal("unknown wrapper should be false")
	}
}
