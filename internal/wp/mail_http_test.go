package wp_test

import (
	"context"
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestMailRecordsWhenBlockExternal(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{BlockExternal: true},
		Mailer: wp.NewMailer(),
	}
	ok, err := wp.Mail(c, []string{"admin@example.com"}, "Hello", "Body", []string{"From: noreply@example.com"})
	if err != nil || !ok {
		t.Fatalf("Mail: ok=%v err=%v", ok, err)
	}
	sent := c.Mailer.Sent()
	if len(sent) != 1 {
		t.Fatalf("sent=%d", len(sent))
	}
	if sent[0].Subject != "Hello" || sent[0].To[0] != "admin@example.com" {
		t.Fatalf("unexpected message: %+v", sent[0])
	}
}

func TestRemoteGetBlocked(t *testing.T) {
	c := &wp.Context{Config: &config.Config{BlockExternal: true}}
	resp, err := wp.RemoteGet(c, "https://example.com/")
	if resp != nil {
		t.Fatalf("expected nil response, got %#v", resp)
	}
	he, ok := err.(*wp.HTTPError)
	if !ok {
		t.Fatalf("err type %T: %v", err, err)
	}
	if he.Code != "http_request_not_executed" {
		t.Fatalf("code=%q", he.Code)
	}
	if !strings.Contains(he.Message, "blocked") {
		t.Fatalf("message=%q", he.Message)
	}
}

func TestRemoteGetAllowedStubPath(t *testing.T) {
	// BlockExternal false: may hit the network; use a guaranteed-invalid host
	// only to assert the block path is not taken (error code differs).
	c := &wp.Context{Config: &config.Config{BlockExternal: false}}
	_, err := wp.RemoteGet(c, "http://127.0.0.1:1/")
	if err == nil {
		t.Fatal("expected connection error when not blocked")
	}
	he, ok := err.(*wp.HTTPError)
	if !ok {
		t.Fatalf("err type %T", err)
	}
	if he.Code == "http_request_not_executed" {
		t.Fatal("should not report blocked when BlockExternal=false")
	}
}

func TestPreWPMailShortCircuit(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{BlockExternal: true},
		Mailer: wp.NewMailer(),
		Hooks:  wp.NewHooks(),
	}
	c.Hooks.AddFilter("pre_wp_mail", func(value any, args ...any) any {
		return false
	}, 10, 2)
	ok, err := wp.Mail(c, []string{"admin@example.com"}, "Hello", "Body", nil)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if ok {
		t.Fatal("expected pre_wp_mail to short-circuit with false")
	}
	if len(c.Mailer.Sent()) != 0 {
		t.Fatalf("short-circuit should not record mail, got %#v", c.Mailer.Sent())
	}
}

func TestWPMailFilterMutatesAttrs(t *testing.T) {
	c := &wp.Context{
		Config: &config.Config{BlockExternal: true},
		Mailer: wp.NewMailer(),
		Hooks:  wp.NewHooks(),
	}
	c.Hooks.AddFilter("wp_mail", func(value any, args ...any) any {
		atts, ok := value.(wp.MailAttrs)
		if !ok {
			t.Fatalf("unexpected value %T", value)
		}
		atts.Subject = "Filtered"
		atts.To = []string{"filtered@example.com"}
		return atts
	}, 10, 1)
	ok, err := wp.Mail(c, []string{"admin@example.com"}, "Hello", "Body", nil)
	if err != nil || !ok {
		t.Fatalf("Mail: ok=%v err=%v", ok, err)
	}
	sent := c.Mailer.Sent()
	if len(sent) != 1 || sent[0].Subject != "Filtered" || sent[0].To[0] != "filtered@example.com" {
		t.Fatalf("unexpected sent: %#v", sent)
	}
}

func TestMailFromContentTypeFiltersAndAttachments(t *testing.T) {
	// P2.98: parse From/Content-Type headers; apply filters; record attachments.
	c := &wp.Context{
		Config:  &config.Config{BlockExternal: true},
		Mailer:  wp.NewMailer(),
		Hooks:   wp.NewHooks(),
		Options: nil,
	}
	c.Hooks.AddFilter("wp_mail_from", func(value any, args ...any) any {
		return "filtered@example.com"
	}, 10, 1)
	c.Hooks.AddFilter("wp_mail_from_name", func(value any, args ...any) any {
		return "Filtered Name"
	}, 10, 1)
	c.Hooks.AddFilter("wp_mail_content_type", func(value any, args ...any) any {
		return "text/html"
	}, 10, 1)
	c.Hooks.AddFilter("wp_mail_charset", func(value any, args ...any) any {
		return "UTF-8"
	}, 10, 1)

	ok, err := wp.MailFull(c,
		[]string{"admin@example.com"},
		"Hello",
		"<p>Body</p>",
		[]string{`From: Site Admin <noreply@example.com>`, `Content-Type: text/plain; charset=ISO-8859-1`},
		[]string{"/tmp/export.zip"},
	)
	if err != nil || !ok {
		t.Fatalf("MailFull: ok=%v err=%v", ok, err)
	}
	sent := c.Mailer.Sent()
	if len(sent) != 1 {
		t.Fatalf("sent=%d", len(sent))
	}
	msg := sent[0]
	if msg.FromEmail != "filtered@example.com" || msg.FromName != "Filtered Name" {
		t.Fatalf("from: email=%q name=%q", msg.FromEmail, msg.FromName)
	}
	if msg.ContentType != "text/html" || msg.Charset != "UTF-8" {
		t.Fatalf("type/charset: %q / %q", msg.ContentType, msg.Charset)
	}
	if len(msg.Attachments) != 1 || msg.Attachments[0] != "/tmp/export.zip" {
		t.Fatalf("attachments: %#v", msg.Attachments)
	}
}

func TestParseMailHeadersFromContentType(t *testing.T) {
	c := &wp.Context{Config: &config.Config{BlockExternal: true}, Mailer: wp.NewMailer()}
	ok, err := wp.Mail(c, []string{"a@b.c"}, "t", "m", []string{
		`From: "Ada Lovelace" <ada@example.com>`,
		"Content-Type: text/html; charset=UTF-8",
	})
	if err != nil || !ok {
		t.Fatalf("Mail: %v %v", ok, err)
	}
	msg := c.Mailer.Sent()[0]
	if msg.FromEmail != "ada@example.com" || msg.FromName != "Ada Lovelace" {
		t.Fatalf("from parse: %+v", msg)
	}
	if msg.ContentType != "text/html" || msg.Charset != "UTF-8" {
		t.Fatalf("content-type parse: %+v", msg)
	}
}

func TestBootWiresMailer(t *testing.T) {
	cfg := &config.Config{
		DBName: "gopress_go", DBUser: "gopress", DBPassword: "gopress",
		DBHost: "127.0.0.1:3306", DBCharset: "utf8mb4", TablePrefix: "wp_",
		ABSPATH: "/workspace/", BlockExternal: true,
		AuthKey: "k", SecureAuthKey: "k", LoggedInKey: "k", NonceKey: "k",
		AuthSalt: "s", SecureAuthSalt: "s", LoggedInSalt: "s", NonceSalt: "s",
	}
	c, err := wp.Boot(context.Background(), cfg)
	if err != nil {
		t.Skipf("Boot unavailable: %v", err)
	}
	defer c.DB.Close()
	if c.Mailer == nil {
		t.Fatal("Mailer not wired")
	}
	ok, err := wp.Mail(c, []string{"a@b.c"}, "t", "m", nil)
	if !ok || err != nil {
		t.Fatalf("Mail after boot: %v %v", ok, err)
	}
	if len(c.Mailer.Sent()) != 1 {
		t.Fatal("expected recorded mail")
	}
}
