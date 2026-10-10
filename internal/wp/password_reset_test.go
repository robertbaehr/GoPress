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

func TestCheckPasswordResetKeyAndResetPassword(t *testing.T) {
	c := cronBoot(t)
	defer c.DB.Close()
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	origPass := user.Pass
	t.Cleanup(func() {
		if c.DB == nil {
			return
		}
		q, args, err := c.DB.Prepare(
			"UPDATE "+c.DB.Users()+" SET user_pass = %s, user_activation_key = %s WHERE ID = %d",
			origPass, "", user.ID,
		)
		if err == nil {
			_, _ = c.DB.Exec(q, args...)
		}
		if c.Users != nil && c.Users.Cache != nil {
			c.Users.Cache.Delete("1", "users")
		}
	})

	key, err := wp.GetPasswordResetKey(c, user)
	if err != nil {
		t.Fatal(err)
	}
	got, err := wp.CheckPasswordResetKey(c, key, "admin")
	if err != nil || got == nil || got.ID != user.ID {
		t.Fatalf("check: user=%v err=%v", got, err)
	}
	if _, err := wp.CheckPasswordResetKey(c, "not-a-real-key", "admin"); err == nil || err.Error() != "invalid_key" {
		t.Fatalf("expected invalid_key, got %v", err)
	}

	newPass := "GoPress-Reset-Pass-99"
	if err := wp.ResetPassword(c, user, newPass); err != nil {
		t.Fatal(err)
	}
	reloaded, err := c.GetUserBy("login", "admin")
	if err != nil || reloaded == nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ActivationKey != "" {
		t.Fatalf("activation key should be cleared: %q", reloaded.ActivationKey)
	}
	if !wp.CheckPassword(newPass, reloaded.Pass) {
		t.Fatalf("new password did not verify")
	}
}
