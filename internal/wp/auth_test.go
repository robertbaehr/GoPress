package wp_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/robertbaehr/GoPress/internal/config"
	"github.com/robertbaehr/GoPress/internal/wp"
)

type authGolden struct {
	Salts  map[string]string `json:"salts"`
	Hashes map[string]string `json:"hashes"`
	Nonces []struct {
		Action string `json:"action"`
		Nonce  string `json:"nonce"`
		Tick   int64  `json:"tick"`
		Token  string `json:"token"`
		UID    int    `json:"uid"`
	} `json:"nonces"`
	FastHash     string `json:"fast_hash"`
	HashPassword struct {
		Input  string `json:"input"`
		Length int    `json:"length"`
		Prefix string `json:"prefix"`
	} `json:"hash_password"`
	PhpassCheck struct {
		KnownBadHash            bool `json:"known_bad_hash"`
		PasswordMatchesPortable bool `json:"password_matches_portable"`
		WrongRejectsPortable    bool `json:"wrong_rejects_portable"`
		WPCheckPassword         bool `json:"wp_check_password"`
	} `json:"phpass_check"`
	AuthCookie struct {
		UserID     int               `json:"user_id"`
		Expiration int64             `json:"expiration"`
		Token      string            `json:"token"`
		Cookies    map[string]string `json:"cookies"`
	} `json:"auth_cookie"`
}

func loadAuthGolden(t *testing.T) authGolden {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	data, err := os.ReadFile(filepath.Join(root, "tests", "golden", "auth", "vectors.json"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var g authGolden
	if err := json.Unmarshal(data, &g); err != nil {
		t.Fatalf("json: %v", err)
	}
	return g
}

func testAuth(t *testing.T) *wp.Auth {
	t.Helper()
	// Fixed oracle salts from wp-config.php / tools/oracle/wp-config.php.tmpl.
	cfg := &config.Config{
		AuthKey:        "gopress-ref-auth-key-0001-fixed-for-oracle",
		SecureAuthKey:  "gopress-ref-secure-auth-key-0002-fixed",
		LoggedInKey:    "gopress-ref-logged-in-key-0003-fixed",
		NonceKey:       "gopress-ref-nonce-key-0004-fixed-oracle",
		AuthSalt:       "gopress-ref-auth-salt-0005-fixed-oracle",
		SecureAuthSalt: "gopress-ref-secure-auth-salt-0006-fixed",
		LoggedInSalt:   "gopress-ref-logged-in-salt-0007-fixed",
		NonceSalt:      "gopress-ref-nonce-salt-0008-fixed-oracle",
	}
	return wp.NewAuth(cfg)
}

func TestAuthSaltsAndHashes(t *testing.T) {
	g := loadAuthGolden(t)
	a := testAuth(t)
	matched, total := 0, 0
	for scheme, want := range g.Salts {
		total++
		got := a.Salt(scheme)
		if got == want {
			matched++
		} else {
			t.Errorf("salt %s: want %q got %q", scheme, want, got)
		}
	}
	for scheme, want := range g.Hashes {
		total++
		got := a.Hash("message", scheme)
		if got == want {
			matched++
		} else {
			t.Errorf("hash %s: want %q got %q", scheme, want, got)
		}
	}
	t.Logf("golden auth salts+hashes: %d/%d match", matched, total)
}

func TestAuthNonces(t *testing.T) {
	g := loadAuthGolden(t)
	a := testAuth(t)
	matched, total := 0, 0
	for _, n := range g.Nonces {
		total++
		got := a.CreateNonce(n.Tick, n.Action, n.UID, n.Token)
		if got == n.Nonce {
			matched++
		} else {
			t.Errorf("nonce tick=%d action=%s uid=%d: want %q got %q", n.Tick, n.Action, n.UID, n.Nonce, got)
		}
	}
	t.Logf("golden auth nonces: %d/%d match", matched, total)
}

func TestAuthFastHash(t *testing.T) {
	g := loadAuthGolden(t)
	got := wp.FastHash("message")
	if got != g.FastHash {
		t.Fatalf("fast_hash: want %q got %q", g.FastHash, got)
	}
	if !wp.VerifyFastHash("message", g.FastHash) {
		t.Fatal("VerifyFastHash failed")
	}
}

func TestAuthHashPassword(t *testing.T) {
	g := loadAuthGolden(t)
	h, err := wp.HashPassword(g.HashPassword.Input)
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != g.HashPassword.Length {
		t.Errorf("length: want %d got %d (%q)", g.HashPassword.Length, len(h), h)
	}
	if len(h) < 3 || h[:3] != g.HashPassword.Prefix {
		t.Errorf("prefix: want %q got %q", g.HashPassword.Prefix, h)
	}
	if !wp.CheckPassword(g.HashPassword.Input, h) {
		t.Fatal("CheckPassword failed for freshly hashed password")
	}
}

func TestAuthPhpassCheck(t *testing.T) {
	g := loadAuthGolden(t)
	portable := wp.HashPasswordPortable("password", 8)
	gotMatch := wp.CheckPassword("password", portable)
	gotWrong := wp.CheckPassword("wrong-password", portable)
	gotBad := wp.CheckPassword("password", "$P$Bxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")

	if gotMatch != g.PhpassCheck.PasswordMatchesPortable {
		t.Errorf("password_matches_portable: want %v got %v", g.PhpassCheck.PasswordMatchesPortable, gotMatch)
	}
	// Golden field is the raw CheckPassword result for wrong password (false).
	if gotWrong != g.PhpassCheck.WrongRejectsPortable {
		t.Errorf("wrong_rejects_portable: want %v got %v", g.PhpassCheck.WrongRejectsPortable, gotWrong)
	}
	if gotBad != g.PhpassCheck.KnownBadHash {
		t.Errorf("known_bad_hash: want %v got %v", g.PhpassCheck.KnownBadHash, gotBad)
	}
	if gotMatch != g.PhpassCheck.WPCheckPassword {
		t.Errorf("wp_check_password: want %v got %v", g.PhpassCheck.WPCheckPassword, gotMatch)
	}
}

func TestAuthCookie(t *testing.T) {
	g := loadAuthGolden(t)
	a := testAuth(t)
	// Fixture admin password hash from oracle site.sql / live DB.
	userPass := "$wp$2y$10$6w0zN/qvwp7cHDgzOIfBPuqucz7bjufoiKA/CAk0RUJ3C3XA3YE/y"
	matched, total := 0, 0
	for scheme, want := range g.AuthCookie.Cookies {
		total++
		got := a.GenerateAuthCookie("admin", userPass, g.AuthCookie.Expiration, scheme, g.AuthCookie.Token)
		if got == want {
			matched++
		} else {
			t.Errorf("cookie %s:\n want %s\n got  %s", scheme, want, got)
		}
	}
	t.Logf("golden auth cookies: %d/%d match", matched, total)
}

func TestAuthGoldenMatchRate(t *testing.T) {
	// Aggregate match rate across salts, hashes, nonces, fast_hash, cookies.
	g := loadAuthGolden(t)
	a := testAuth(t)
	matched, total := 0, 0

	check := func(ok bool) {
		total++
		if ok {
			matched++
		}
	}
	for scheme, want := range g.Salts {
		check(a.Salt(scheme) == want)
	}
	for scheme, want := range g.Hashes {
		check(a.Hash("message", scheme) == want)
	}
	for _, n := range g.Nonces {
		check(a.CreateNonce(n.Tick, n.Action, n.UID, n.Token) == n.Nonce)
	}
	check(wp.FastHash("message") == g.FastHash)

	userPass := "$wp$2y$10$6w0zN/qvwp7cHDgzOIfBPuqucz7bjufoiKA/CAk0RUJ3C3XA3YE/y"
	for scheme, want := range g.AuthCookie.Cookies {
		check(a.GenerateAuthCookie("admin", userPass, g.AuthCookie.Expiration, scheme, g.AuthCookie.Token) == want)
	}

	h, err := wp.HashPassword("password")
	if err != nil {
		t.Fatal(err)
	}
	check(len(h) == g.HashPassword.Length && len(h) >= 3 && h[:3] == g.HashPassword.Prefix)
	check(wp.CheckPassword("password", h))

	portable := wp.HashPasswordPortable("password", 8)
	check(wp.CheckPassword("password", portable) == g.PhpassCheck.PasswordMatchesPortable)
	check(wp.CheckPassword("wrong-password", portable) == g.PhpassCheck.WrongRejectsPortable)
	check(wp.CheckPassword("password", "$P$Bxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx") == g.PhpassCheck.KnownBadHash)

	t.Logf("golden auth overall: %d/%d cells match (%.1f%%)", matched, total, 100*float64(matched)/float64(total))
	if matched != total {
		t.Errorf("incomplete match: %d/%d", matched, total)
	}
}
