package wp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSendPersonalDataExportEmail(t *testing.T) {
	c := &wp.Context{
		Hooks:   wp.NewHooks(),
		Mailer:  wp.NewMailer(),
		Options: wp.NewOptions(nil, wp.NewObjectCache()),
	}
	c.Options.Cache.Set("blogname", "Export Site", "options")
	c.Options.Cache.Set("home", "http://example.com", "options")
	c.Options.Cache.Set("siteurl", "http://example.com", "options")
	c.Options.Cache.Set("date_format", "F j, Y", "options")

	err := wp.SendPersonalDataExportEmail(c, "user@example.com", "http://example.com/exports/file.html")
	if err != nil {
		t.Fatal(err)
	}
	sent := c.Mailer.Sent()
	if len(sent) != 1 {
		t.Fatalf("sent=%d", len(sent))
	}
	msg := sent[0]
	if len(msg.To) != 1 || msg.To[0] != "user@example.com" {
		t.Fatalf("to=%v", msg.To)
	}
	if !strings.Contains(msg.Subject, "Personal Data Export") || !strings.Contains(msg.Subject, "Export Site") {
		t.Fatalf("subject=%q", msg.Subject)
	}
	for _, want := range []string{
		"http://example.com/exports/file.html",
		"Export Site",
		"http://example.com",
		"Howdy,",
	} {
		if !strings.Contains(msg.Message, want) {
			t.Fatalf("message missing %q: %q", want, msg.Message)
		}
	}
}

func TestSendPersonalDataExportEmailRejectsBadInput(t *testing.T) {
	c := &wp.Context{Mailer: wp.NewMailer()}
	if err := wp.SendPersonalDataExportEmail(c, "bad", "http://x"); err == nil {
		t.Fatal("expected bad email error")
	}
	if err := wp.SendPersonalDataExportEmail(c, "a@b.com", ""); err == nil {
		t.Fatal("expected missing URL error")
	}
}

func TestSendPersonalDataExportEmailForResult(t *testing.T) {
	c := testBootQuery(t)
	if c.Mailer == nil {
		c.Mailer = wp.NewMailer()
	}
	c.Mailer.Clear()
	res, err := wp.GeneratePersonalDataExportFile(c, "export-mail@example.com")
	if err != nil {
		t.Skipf("export file unavailable: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(res.JSONPath)
		_ = os.Remove(res.HTMLPath)
		_ = os.Remove(filepath.Join(filepath.Dir(res.JSONPath), "index.php"))
	})
	if err := wp.SendPersonalDataExportEmailForResult(c, "export-mail@example.com", res); err != nil {
		t.Fatal(err)
	}
	sent := c.Mailer.Sent()
	if len(sent) == 0 {
		t.Fatal("expected mail")
	}
	last := sent[len(sent)-1]
	if len(last.To) == 0 || last.To[0] != "export-mail@example.com" {
		t.Fatalf("unexpected mail: %+v", last)
	}
}
