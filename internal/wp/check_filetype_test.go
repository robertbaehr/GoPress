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
