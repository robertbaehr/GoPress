package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestConvertHrToBytes(t *testing.T) {
	cases := []struct {
		in   string
		want int64
	}{
		{"64M", 64 * 1024 * 1024},
		{"1G", 1024 * 1024 * 1024},
		{"512K", 512 * 1024},
		{"100", 100},
		{"", 0},
	}
	for _, tc := range cases {
		if got := wp.ConvertHrToBytes(tc.in); got != tc.want {
			t.Fatalf("%q → %d want %d", tc.in, got, tc.want)
		}
	}
}

func TestMaxUploadSizeAndPluploadSettings(t *testing.T) {
	t.Setenv("PHP_UPLOAD_MAX_FILESIZE", "8M")
	t.Setenv("PHP_POST_MAX_SIZE", "16M")
	size := wp.MaxUploadSize(nil)
	if size != 8*1024*1024 {
		t.Fatalf("MaxUploadSize=%d", size)
	}

	c := wp.NewContext(nil)
	c.Hooks = wp.NewHooks()
	settings := wp.PluploadDefaultSettings(c)
	if settings.Defaults["file_data_name"] != "async-upload" {
		t.Fatalf("file_data_name=%v", settings.Defaults["file_data_name"])
	}
	if settings.Defaults["url"] != "/wp-admin/async-upload.php" {
		t.Fatalf("url=%v", settings.Defaults["url"])
	}
	filters, _ := settings.Defaults["filters"].(map[string]any)
	if filters["max_file_size"] != "8388608b" {
		t.Fatalf("max_file_size=%v", filters["max_file_size"])
	}
	params, _ := settings.Defaults["multipart_params"].(map[string]any)
	if params["action"] != "upload-attachment" {
		t.Fatalf("action=%v", params["action"])
	}
	if _, ok := params["_wpnonce"].(string); !ok {
		t.Fatalf("nonce missing: %v", params)
	}
	mimeTypes, _ := filters["mime_types"].([]map[string]any)
	if len(mimeTypes) != 1 {
		t.Fatalf("mime_types=%v", filters["mime_types"])
	}
	exts, _ := mimeTypes[0]["extensions"].(string)
	if !strings.Contains(exts, "jpg") && !strings.Contains(exts, "jpeg") {
		t.Fatalf("extensions=%q", exts)
	}
	if settings.Browser["supported"] != true {
		t.Fatalf("browser=%v", settings.Browser)
	}
	if settings.Defaults["webp_upload_error"] != true {
		t.Fatal("expected webp_upload_error")
	}
}

func TestPluploadDefaultSettingsFilter(t *testing.T) {
	c := wp.NewContext(nil)
	c.Hooks = wp.NewHooks()
	c.Hooks.AddFilter("plupload_default_settings", func(value any, _ ...any) any {
		m, _ := value.(map[string]any)
		m["multi_selection"] = false
		return m
	}, 10, 1)
	settings := wp.PluploadDefaultSettings(c)
	if settings.Defaults["multi_selection"] != false {
		t.Fatalf("filter not applied: %v", settings.Defaults["multi_selection"])
	}
}
