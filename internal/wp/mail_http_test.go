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
