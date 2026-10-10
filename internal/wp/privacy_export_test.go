package wp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
