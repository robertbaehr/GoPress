package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestGeneratePasswordLength(t *testing.T) {
	p := wp.GeneratePassword(20, false)
	if len(p) != 20 {
		t.Fatalf("len=%d", len(p))
	}
	for _, r := range p {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			t.Fatalf("unexpected char %q in %q", r, p)
		}
	}
}

func TestGetPasswordResetKeyStoresActivation(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	key, err := wp.GetPasswordResetKey(c, user)
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 20 {
		t.Fatalf("key len=%d", len(key))
	}
	reloaded, err := c.GetUserBy("id", user.ID)
	if err != nil || reloaded == nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ActivationKey == "" || !strings.Contains(reloaded.ActivationKey, ":") {
		t.Fatalf("activation key=%q", reloaded.ActivationKey)
	}
	if !strings.HasPrefix(strings.SplitN(reloaded.ActivationKey, ":", 2)[1], "$generic$") {
		t.Fatalf("expected fast_hash in activation key: %q", reloaded.ActivationKey)
	}
}

func TestRetrievePasswordSendsMail(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()
	if c.Mailer == nil {
		c.Mailer = wp.NewMailer()
	}
	c.Mailer.Clear()
	if err := wp.RetrievePassword(c, "admin"); err != nil {
		t.Fatal(err)
	}
	sent := c.Mailer.Sent()
	if len(sent) != 1 {
		t.Fatalf("sent=%d", len(sent))
	}
	if !strings.Contains(sent[0].Subject, "Password Reset") {
		t.Fatalf("subject=%q", sent[0].Subject)
	}
	if !strings.Contains(sent[0].Message, "action=rp") || !strings.Contains(sent[0].Message, "key=") {
		t.Fatalf("message missing reset link: %q", sent[0].Message)
	}
}
