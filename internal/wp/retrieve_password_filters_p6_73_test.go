package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

// P6.73: retrieve_password_title / retrieve_password_message filter the reset email.
func TestRetrievePasswordFiltersP673(t *testing.T) {
	c := testBootQuery(t)
	if c.Mailer == nil {
		c.Mailer = wp.NewMailer()
	}
	c.Mailer.Clear()

	c.Hooks.AddFilter("retrieve_password_title", func(value any, args ...any) any {
		return "Custom Reset Title"
	}, 10, 3)
	c.Hooks.AddFilter("retrieve_password_message", func(value any, args ...any) any {
		msg, _ := value.(string)
		key := ""
		if len(args) >= 1 {
			key, _ = args[0].(string)
		}
		return "Custom body key=" + key + "\n" + msg
	}, 10, 4)

	if err := wp.RetrievePassword(c, "admin"); err != nil {
		t.Fatal(err)
	}
	sent := c.Mailer.Sent()
	if len(sent) != 1 {
		t.Fatalf("sent=%d", len(sent))
	}
	if sent[0].Subject != "Custom Reset Title" {
		t.Fatalf("subject=%q", sent[0].Subject)
	}
	if !strings.HasPrefix(sent[0].Message, "Custom body key=") {
		t.Fatalf("message=%q", sent[0].Message)
	}
	if !strings.Contains(sent[0].Message, "Password Reset") && !strings.Contains(sent[0].Message, "reset") {
		// Original body still appended after custom prefix.
		if !strings.Contains(sent[0].Message, "wp-login.php") {
			t.Fatalf("expected original body remnant: %q", sent[0].Message)
		}
	}
}
