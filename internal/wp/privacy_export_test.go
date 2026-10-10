package wp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGeneratePersonalDataExportFile(t *testing.T) {
	c := testBootQuery(t)

	res, err := wp.GeneratePersonalDataExportFile(c, "privacy@example.com")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(res.JSONPath)
		_ = os.Remove(res.HTMLPath)
	})
	if !strings.HasSuffix(res.JSONPath, ".json") || !strings.HasSuffix(res.HTMLPath, ".html") {
		t.Fatalf("paths: %+v", res)
	}
	raw, err := os.ReadFile(res.JSONPath)
	if err != nil {
		t.Fatal(err)
	}
	var report wp.PersonalDataExport
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if report.Email != "privacy@example.com" {
		t.Fatalf("email=%q", report.Email)
	}
	if len(report.Groups) == 0 {
		t.Fatal("expected about group")
	}
	html, err := os.ReadFile(res.HTMLPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(html), "privacy@example.com") {
		t.Fatalf("html missing email: %s", html)
	}
	index := filepath.Join(filepath.Dir(res.JSONPath), "index.php")
	if _, err := os.Stat(index); err != nil {
		t.Fatalf("expected index.php guard: %v", err)
	}
}

func TestGeneratePersonalDataExportFileRejectsBadEmail(t *testing.T) {
	c := testBootQuery(t)
	if _, err := wp.GeneratePersonalDataExportFile(c, "not-an-email"); err == nil {
		t.Fatal("expected error")
	}
}

func TestPersonalDataExportDownloadExpires(t *testing.T) {
	c := testBootQuery(t)
	res, err := wp.GeneratePersonalDataExportFile(c, "expire@example.com")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Remove(res.JSONPath)
		_ = os.Remove(res.HTMLPath)
	})
	base := filepath.Base(res.HTMLPath)
	if got := wp.ResolvePersonalDataExportFile(c, base); got == "" {
		t.Fatal("fresh export should resolve")
	}

	// Force mtime into the past beyond default 3-day expiration.
	past := time.Now().Add(-4 * 24 * time.Hour)
	if err := os.Chtimes(res.HTMLPath, past, past); err != nil {
		t.Fatal(err)
	}
	if !wp.PersonalDataExportExpired(c, res.HTMLPath) {
		t.Fatal("expected expired")
	}
	if got := wp.ResolvePersonalDataExportFile(c, base); got != "" {
		t.Fatalf("expired export should not resolve, got %q", got)
	}
	if _, err := os.Stat(res.HTMLPath); !os.IsNotExist(err) {
		t.Fatalf("expired file should be deleted on resolve, err=%v", err)
	}
}

func TestDeleteExpiredPersonalDataExports(t *testing.T) {
	c := testBootQuery(t)
	res, err := wp.GeneratePersonalDataExportFile(c, "cleanup@example.com")
	if err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-10 * 24 * time.Hour)
	_ = os.Chtimes(res.JSONPath, past, past)
	_ = os.Chtimes(res.HTMLPath, past, past)
	n := wp.DeleteExpiredPersonalDataExports(c)
	if n < 2 {
		t.Fatalf("expected ≥2 deletions, got %d", n)
	}
	if _, err := os.Stat(res.HTMLPath); !os.IsNotExist(err) {
		t.Fatal("html should be gone")
	}
}

func TestPrivacyExportExpirationFilter(t *testing.T) {
	c := testBootQuery(t)
	if c.Hooks == nil {
		c.Hooks = wp.NewHooks()
	}
	c.Hooks.AddFilter("wp_privacy_export_expiration", func(value any, _ ...any) any {
		return 60
	}, 10, 1)
	if got := wp.PrivacyExportExpiration(c); got != 60 {
		t.Fatalf("expiration=%d want 60", got)
	}
}
