package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestSendPersonalDataErasureEmail(t *testing.T) {
	c := &wp.Context{
		Hooks:   wp.NewHooks(),
		Mailer:  wp.NewMailer(),
		Options: wp.NewOptions(nil, wp.NewObjectCache()),
	}
	c.Options.Cache.Set("blogname", "Erase Site", "options")
	c.Options.Cache.Set("home", "http://example.com", "options")
	c.Options.Cache.Set("siteurl", "http://example.com", "options")

	err := wp.SendPersonalDataErasureEmail(c, "user@example.com")
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
	if !strings.Contains(msg.Subject, "Erasure Request Fulfilled") || !strings.Contains(msg.Subject, "Erase Site") {
		t.Fatalf("subject=%q", msg.Subject)
	}
	for _, want := range []string{
		"Erase Site",
		"http://example.com",
		"Howdy,",
		"has been completed",
	} {
		if !strings.Contains(msg.Message, want) {
			t.Fatalf("message missing %q: %q", want, msg.Message)
		}
	}
}

func TestSendPersonalDataErasureEmailRejectsBadInput(t *testing.T) {
	c := &wp.Context{Mailer: wp.NewMailer()}
	if err := wp.SendPersonalDataErasureEmail(c, "bad"); err == nil {
		t.Fatal("expected bad email error")
	}
}
