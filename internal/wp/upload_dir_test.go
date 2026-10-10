package wp_test

import (
	"strings"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGetUploadDirYearMonth(t *testing.T) {
	c := testBootQuery(t)
	when := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	dir := wp.GetUploadDirAt(c, when)
	if dir.Basedir == "" || dir.Baseurl == "" {
		t.Fatalf("empty base: %#v", dir)
	}
	if !strings.HasSuffix(dir.Basedir, "/wp-content/uploads") {
		t.Fatalf("basedir=%q", dir.Basedir)
	}
	if dir.Subdir != "/2024/06" {
		t.Fatalf("subdir=%q want /2024/06", dir.Subdir)
	}
	if dir.Path != dir.Basedir+"/2024/06" {
		t.Fatalf("path=%q basedir=%q", dir.Path, dir.Basedir)
	}
	if dir.URL != dir.Baseurl+"/2024/06" {
		t.Fatalf("url=%q baseurl=%q", dir.URL, dir.Baseurl)
	}
	if dir.Error != "" {
		t.Fatalf("error=%q", dir.Error)
	}
}

func TestGetUploadDirWithoutYearMonth(t *testing.T) {
	c := testBootQuery(t)
	prev := c.Options.GetOption("uploads_use_yearmonth_folders", "1")
	if _, err := c.Options.UpdateOption("uploads_use_yearmonth_folders", "0", "yes"); err != nil {
		t.Fatalf("update option: %v", err)
	}
	t.Cleanup(func() {
		_, _ = c.Options.UpdateOption("uploads_use_yearmonth_folders", prev, "yes")
	})
	dir := wp.GetUploadDir(c)
	if dir.Subdir != "" {
		t.Fatalf("subdir=%q want empty", dir.Subdir)
	}
	if dir.Path != dir.Basedir || dir.URL != dir.Baseurl {
		t.Fatalf("path/url should equal base without subdir: %#v", dir)
	}
}

func TestGetUploadDirAlias(t *testing.T) {
	c := testBootQuery(t)
	a := wp.GetUploadDir(c)
	b := wp.GetUploadDirAt(c, time.Time{})
	if a.Basedir != b.Basedir || a.Baseurl != b.Baseurl {
		t.Fatalf("GetUploadDir mismatch: %#v vs %#v", a, b)
	}
}
