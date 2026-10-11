package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestBasename(t *testing.T) {
	cases := []struct {
		path, suffix, want string
	}{
		{"http://ex.com/a/b%20c.jpg", "", "b%20c.jpg"},
		{"/foo/bar.baz.txt", ".txt", "bar.baz"},
		{`C:\Windows\file.txt`, "", "file.txt"},
		{"a/b/c", "", "c"},
		{"", "", ""},
		{"foo/", "", "foo"},
		{"/foo/bar/", "", "bar"},
		{"%2Fetc%2Fpasswd", "", "%2Fetc%2Fpasswd"},
		{`path%5Cto%5Cfile`, "", `path%5Cto%5Cfile`},
		{"café/naïve.txt", "", "naïve.txt"},
		{"a b/c d.txt", "", "c d.txt"},
		// P2.113 deepen: separators, query, PHP 8+ full-suffix keep.
		{"//", "", ""},
		{"///", "", ""},
		{".", "", "."},
		{"..", "", ".."},
		{"foo/bar.php", ".php", "bar"},
		{"foo/bar.php", ".PHP", "bar.php"}, // case-sensitive suffix
		{"foo/bar.php", "bar.php", "bar.php"},
		{"a/b.jpg?x=1", "", "b.jpg?x=1"},
		{`foo\bar\baz.txt`, ".txt", "baz"},
		{"foo/./bar", "", "bar"},
		{" ", "", " "},
		{"a/b/c.", ".", "c"},
		{"a/b/c.", "", "c."},
	}
	for _, tc := range cases {
		got := wp.Basename(tc.path, tc.suffix)
		if got != tc.want {
			t.Fatalf("Basename(%q, %q)=%q want %q", tc.path, tc.suffix, got, tc.want)
		}
	}
}
