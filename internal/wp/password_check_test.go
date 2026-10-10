package wp_test

import (
	"strings"
	"testing"

	"github.com/robertbaehr/GoPress/internal/wp"
)

func TestWPCheckPasswordCoversPhpassAndWPPrefix(t *testing.T) {
	plain := "password"

	wpHash, err := wp.HashPassword(plain)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(wpHash, "$wp$") {
		t.Fatalf("expected $wp$ hash, got %q", wpHash)
	}
	if !wp.WPCheckPassword(plain, wpHash) {
		t.Fatal("$wp$ hash should verify")
	}
	if wp.WPCheckPassword("wrong-password", wpHash) {
		t.Fatal("$wp$ wrong password should reject")
	}

	portable := wp.HashPasswordPortable(plain, 8)
	if !strings.HasPrefix(portable, "$P$") {
		t.Fatalf("expected $P$ hash, got %q", portable)
	}
	if !wp.WPCheckPassword(plain, portable) {
		t.Fatal("phpass $P$ should verify")
	}
	if wp.WPCheckPassword("wrong-password", portable) {
		t.Fatal("phpass wrong password should reject")
	}

	// $H$ is an alternate phpass portable prefix (same algorithm).
	hPrefix := "$H$" + portable[3:]
	if !wp.CheckPassword(plain, hPrefix) {
		t.Fatal("phpass $H$ should verify")
	}

	// Fixture admin user_pass (oracle) is a $wp$ hash for "password".
	fixture := "$wp$2y$10$6w0zN/qvwp7cHDgzOIfBPuqucz7bjufoiKA/CAk0RUJ3C3XA3YE/y"
	if !wp.WPCheckPassword(plain, fixture) {
		t.Fatal("fixture $wp$ admin hash should verify")
	}
}

func TestApplicationPasswordUsesCheckPasswordFormats(t *testing.T) {
	c := testBootQuery(t)
	user, err := c.GetUserBy("login", "admin")
	if err != nil || user == nil {
		t.Fatalf("admin: %v", err)
	}
	plain := "abcd efgh ijkl mnop qrst uvwx"
	compact := strings.ReplaceAll(plain, " ", "")

	// App passwords commonly use $generic$ fast-hash; VerifyFastHash path.
	fast := wp.FastHash(compact)
	if err := wp.SetApplicationPasswordsForTest(c, user.ID, "", "Fast", fast); err != nil {
		t.Fatal(err)
	}
	if !wp.CheckApplicationPassword(c, user.ID, plain) {
		t.Fatal("expected $generic$ application password match")
	}

	// Also accept phpass / $wp$ hashes when stored as application password passwords.
	portable := wp.HashPasswordPortable(compact, 8)
	if err := wp.SetApplicationPasswordsForTest(c, user.ID, "", "Phpass", portable); err != nil {
		t.Fatal(err)
	}
	if !wp.VerifyApplicationPasswordHash(compact, portable) {
		t.Fatal("application password verifier should accept phpass via CheckPassword")
	}
	if !wp.CheckApplicationPassword(c, user.ID, plain) {
		t.Fatal("expected phpass application password match")
	}

	wpHash, err := wp.HashPassword(compact)
	if err != nil {
		t.Fatal(err)
	}
	if err := wp.SetApplicationPasswordsForTest(c, user.ID, "", "WPBcrypt", wpHash); err != nil {
		t.Fatal(err)
	}
	if !wp.VerifyApplicationPasswordHash(compact, wpHash) {
		t.Fatal("application password verifier should accept $wp$ via CheckPassword")
	}
	if !wp.CheckApplicationPassword(c, user.ID, plain) {
		t.Fatal("expected $wp$ application password match")
	}
}
