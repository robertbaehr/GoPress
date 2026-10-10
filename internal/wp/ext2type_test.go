package wp_test

import (
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestExt2TypeDefaults(t *testing.T) {
	cases := []struct {
		ext  string
		want string
	}{
		{"jpg", "image"},
		{".PNG", "image"},
		{"mp3", "audio"},
		{"MP4", "video"},
		{"pdf", "document"},
		{"xlsx", "spreadsheet"},
		{"pptx", "interactive"},
		{"txt", "text"},
		{"zip", "archive"},
		{"js", "code"},
		{"unknownxyz", ""},
		{"", ""},
	}
	for _, tc := range cases {
		got := wp.Ext2Type(nil, tc.ext)
		if got != tc.want {
			t.Fatalf("Ext2Type(%q)=%q want %q", tc.ext, got, tc.want)
		}
	}
}

func TestGetExtTypesFilter(t *testing.T) {
	c := &wp.Context{Hooks: wp.NewHooks()}
	c.Hooks.AddFilter("ext2type", func(value any, args ...any) any {
		m, _ := value.(map[string][]string)
		m["image"] = append(m["image"], "customimg")
		return m
	}, 10, 1)
	if got := wp.Ext2Type(c, "customimg"); got != "image" {
		t.Fatalf("filtered: got %q", got)
	}
	// Default map without context is unchanged.
	if got := wp.Ext2Type(nil, "customimg"); got != "" {
		t.Fatalf("nil context should ignore filter: %q", got)
	}
}

func TestGetExtTypesCopy(t *testing.T) {
	a := wp.GetExtTypes(nil)
	a["image"] = append(a["image"], "zzz")
	b := wp.GetExtTypes(nil)
	for _, e := range b["image"] {
		if e == "zzz" {
			t.Fatal("GetExtTypes should return a copy")
		}
	}
}
