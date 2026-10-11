package wp_test

import (
	"os"
	"path/filepath"
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

func TestCheckFiletypeNormalizesQueryFragmentAndPath(t *testing.T) {
	cases := []string{
		"/var/uploads/2024/song.mp3",
		"https://cdn.example.com/media/song.mp3?ver=2",
		"song.mp3#t=10",
		`C:\tmp\song.mp3`,
	}
	for _, file := range cases {
		got := wp.CheckFiletype(nil, file, nil)
		if got.Ext != "mp3" || got.Type != "audio/mpeg" {
			t.Fatalf("CheckFiletype(%q)=%+v want mp3/audio/mpeg", file, got)
		}
	}
}

func TestCheckFiletypeStableCustomMimeOrder(t *testing.T) {
	// Overlapping patterns: first key in sorted order among extras wins when
	// neither is in the default order list — "aaa" before "zzz".
	mimes := map[string]string{
		"zzz": "application/z",
		"aaa": "application/a",
	}
	got := wp.CheckFiletype(nil, "file.aaa", mimes)
	if got.Ext != "aaa" || got.Type != "application/a" {
		t.Fatalf("got=%+v", got)
	}
	got = wp.CheckFiletype(nil, "file.zzz", mimes)
	if got.Ext != "zzz" || got.Type != "application/z" {
		t.Fatalf("got=%+v", got)
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
	if _, ok := m["js"]; ok {
		t.Fatal("js should be unset without unfiltered_html")
	}
}

func TestGetAllowedMimeTypesForUserUnfilteredHTML(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks(), Roles: &wp.Roles{}}
	admin := &wp.User{ID: 1, Login: "admin", AllCaps: map[string]bool{"unfiltered_html": true}}
	sub := &wp.User{ID: 2, Login: "sub", AllCaps: map[string]bool{"read": true}}
	c.CurrentUser = sub

	// Explicit privileged user keeps html/js even when current user cannot.
	m := wp.GetAllowedMimeTypes(c, admin)
	if m["htm|html"] == "" || m["js"] == "" {
		t.Fatalf("admin unfiltered_html should keep html/js: %#v", m["htm|html"])
	}
	// Current (subscriber) user path strips them.
	m2 := wp.GetAllowedMimeTypes(c)
	if _, ok := m2["htm|html"]; ok {
		t.Fatal("subscriber should not keep htm|html")
	}

	var sawUser *wp.User
	c.Hooks.AddFilter("upload_mimes", func(value any, args ...any) any {
		if len(args) > 0 {
			if u, ok := args[0].(*wp.User); ok {
				sawUser = u
			}
		}
		return value
	}, 10, 2)
	_ = wp.GetAllowedMimeTypes(c, admin)
	if sawUser == nil || sawUser.ID != admin.ID {
		t.Fatalf("upload_mimes user arg: %#v", sawUser)
	}
}

func TestGetMimeTypesFullParityKeys(t *testing.T) {
	m := wp.GetMimeTypes(nil)
	want := map[string]string{
		"dfxp":                         "application/ttaf+xml",
		"mdb":                          "application/vnd.ms-access",
		"mpp":                          "application/vnd.ms-project",
		"docm":                         "application/vnd.ms-word.document.macroEnabled.12",
		"dotx":                         "application/vnd.openxmlformats-officedocument.wordprocessingml.template",
		"xlsm":                         "application/vnd.ms-excel.sheet.macroEnabled.12",
		"ppsx":                         "application/vnd.openxmlformats-officedocument.presentationml.slideshow",
		"onetoc|onetoc2|onetmp|onepkg": "application/onenote",
		"oxps":                         "application/oxps",
		"xps":                          "application/vnd.ms-xpsdocument",
		"odg":                          "application/vnd.oasis.opendocument.graphics",
		"odc":                          "application/vnd.oasis.opendocument.chart",
		"odb":                          "application/vnd.oasis.opendocument.database",
		"odf":                          "application/vnd.oasis.opendocument.formula",
	}
	for k, v := range want {
		if got := m[k]; got != v {
			t.Fatalf("GetMimeTypes[%q]=%q want %q", k, got, v)
		}
	}
	got := wp.CheckFiletype(nil, "notes.dfxp", m)
	if got.Ext != "dfxp" || got.Type != "application/ttaf+xml" {
		t.Fatalf("CheckFiletype dfxp=%+v", got)
	}
}

func TestGetMimeTypesFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("mime_types", func(value any, _ ...any) any {
		m := value.(map[string]string)
		m["xyz"] = "application/x-xyz"
		return m
	}, 10, 1)
	m := wp.GetMimeTypes(c)
	if m["xyz"] != "application/x-xyz" {
		t.Fatalf("mime_types filter not applied: %+v", m["xyz"])
	}
}

func TestCheckFiletypeAndExtBasic(t *testing.T) {
	got := wp.CheckFiletypeAndExt(nil, "", "photo.PNG", nil)
	if got.Ext != "png" || got.Type != "image/png" {
		t.Fatalf("got=%+v", got)
	}
}

func TestCheckFiletypeAndExtCorrectsImageExt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "misnamed.gif")
	// Minimal JPEG magic.
	data := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F'}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	got := wp.CheckFiletypeAndExt(nil, path, "misnamed.gif", nil)
	if got.Type != "image/jpeg" {
		t.Fatalf("type=%q want image/jpeg (%+v)", got.Type, got)
	}
	if got.ProperFilename != "misnamed.jpg" && got.Ext != "jpg" && got.Ext != "jpeg" {
		t.Fatalf("expected corrected filename/ext: %+v", got)
	}
}

func TestCheckFiletypeAndExtHonorsUploadMimes(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("upload_mimes", func(value any, _ ...any) any {
		return map[string]string{"custom": "application/x-custom"}
	}, 10, 2)
	ok := wp.CheckFiletypeAndExt(c, "", "a.custom", nil)
	if ok.Ext != "custom" || ok.Type != "application/x-custom" {
		t.Fatalf("custom mime: %+v", ok)
	}
	deny := wp.CheckFiletypeAndExt(c, "", "a.png", nil)
	if deny.Ext != "" || deny.Type != "" {
		t.Fatalf("png should be denied when upload_mimes narrowed: %+v", deny)
	}
}
