package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestCheckFiletypeDefaults(t *testing.T) {
	cases := []struct {
		file string
		ext  string
		mime string
	}{
		{"photo.JPG", "jpg", "image/jpeg"},
		{"/tmp/a.png", "png", "image/png"},
		{"doc.pdf", "pdf", "application/pdf"},
		{"song.mp3", "mp3", "audio/mpeg"},
		{"clip.mp4", "mp4", "video/mp4"},
		{"noext", "", ""},
		{"bad.exe", "", ""},   // exe removed from allowed
		{"page.html", "", ""}, // html removed without unfiltered_html
	}
	for _, tc := range cases {
		got := wp.CheckFiletype(nil, tc.file, nil)
		if got.Ext != tc.ext || got.Type != tc.mime {
			t.Fatalf("CheckFiletype(%q)=%+v want ext=%q type=%q", tc.file, got, tc.ext, tc.mime)
		}
	}
}

func TestCheckFiletypeCustomMimes(t *testing.T) {
	mimes := map[string]string{"foo|bar": "application/x-foo"}
	got := wp.CheckFiletype(nil, "x.Bar", mimes)
	if got.Ext != "bar" || got.Type != "application/x-foo" {
		t.Fatalf("%+v", got)
	}
}

func TestGetAllowedMimeTypesUnsets(t *testing.T) {
	m := wp.GetAllowedMimeTypes(nil)
	if _, ok := m["swf"]; ok {
		t.Fatal("swf should be unset")
	}
	if _, ok := m["exe"]; ok {
		t.Fatal("exe should be unset")
	}
	if _, ok := m["htm|html"]; ok {
		t.Fatal("htm|html should be unset without unfiltered_html")
	}
}
